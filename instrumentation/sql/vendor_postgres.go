package sql

import (
	"fmt"
	"net"
	nurl "net/url"
	"sort"
	"strings"
	"unicode"
)

type postgresExtension struct{}

// scanner implements a tokenizer for libpq-style option strings.
type scanner struct {
	s []rune
	i int
}

func init() {
	vendorExtensions = append(vendorExtensions, &postgresExtension{})
}

// Gets if the extension is compatible with the component name
func (ext *postgresExtension) IsCompatible(componentName string) bool {
	return componentName == "pq.Driver" || componentName == "stdlib.Driver" || componentName == "pgsqldriver.postgresDriver"
}

// Complete the missing driver data from the connection string
func (ext *postgresExtension) ProcessConnectionString(connectionString string, configuration *driverConfiguration) {
	configuration.peerService = "postgresql"

	dsn := connectionString
	if strings.HasPrefix(connectionString, "postgres://") || strings.HasPrefix(connectionString, "postgresql://") {
		if pDsn, err := ext.parseUrl(connectionString); err == nil {
			dsn = pDsn
		}
	}
	o := make(values)
	o["host"] = "localhost"
	o["port"] = "5432"
	_ = ext.parseOpts(dsn, o)
	o["password"] = "******"

	if user, ok := o["user"]; ok {
		configuration.user = user
	}
	if port, ok := o["port"]; ok {
		configuration.port = port
	}
	if dbname, ok := o["dbname"]; ok {
		configuration.instance = dbname
	}
	if host, ok := o["host"]; ok {
		configuration.host = host
	}

	cStringBuilder := strings.Builder{}
	for key, value := range o {
		cStringBuilder.WriteString(fmt.Sprintf("%v=%v ", key, value))
	}
	configuration.connString = strings.TrimSpace(cStringBuilder.String())
}

// postgress ParseURL no longer needs to be used by clients of this library since supplying a URL as a
// connection string to sql.Open() is now supported:
//
//	sql.Open("postgres", "postgres://bob:secret@1.2.3.4:5432/mydb?sslmode=verify-full")
//
// It remains exported here for backwards-compatibility.
//
// ParseURL converts a url to a connection string for driver.Open.
// Example:
//
//	"postgres://bob:secret@1.2.3.4:5432/mydb?sslmode=verify-full"
//
// converts to:
//
//	"user=bob password=secret host=1.2.3.4 port=5432 dbname=mydb sslmode=verify-full"
//
// A minimal example:
//
//	"postgres://"
//
// This will be blank, causing driver.Open to use all of the defaults
func (ext *postgresExtension) parseUrl(url string) (string, error) {
	u, err := nurl.Parse(url)
	if err != nil {
		return "", err
	}

	if u.Scheme != "postgres" && u.Scheme != "postgresql" {
		return "", fmt.Errorf("invalid connection protocol: %s", u.Scheme)
	}

	var kvs []string
	escaper := strings.NewReplacer(` `, `\ `, `'`, `\'`, `\`, `\\`)
	accrue := func(k, v string) {
		if v != "" {
			kvs = append(kvs, k+"="+escaper.Replace(v))
		}
	}

	if u.User != nil {
		v := u.User.Username()
		accrue("user", v)

		v, _ = u.User.Password()
		accrue("password", v)
	}

	if host, port, err := net.SplitHostPort(u.Host); err != nil {
		accrue("host", u.Host)
	} else {
		accrue("host", host)
		accrue("port", port)
	}

	if u.Path != "" {
		accrue("dbname", u.Path[1:])
	}

	q := u.Query()
	for k := range q {
		accrue(k, q.Get(k))
	}

	sort.Strings(kvs) // Makes testing easier (not a performance concern)
	return strings.Join(kvs, " "), nil
}

// parseOpts parses the options from name and adds them to the values.
//
// The parsing code is based on conninfo_parse from libpq's fe-connect.c
func (ext *postgresExtension) parseOpts(name string, o values) error {
	s := ext.newScanner(name)

	for {
		var (
			keyRunes, valRunes []rune
			r                  rune
			ok                 bool
		)

		if r, ok = s.SkipSpaces(); !ok {
			break
		}

		// Scan the key
		for !unicode.IsSpace(r) && r != '=' {
			keyRunes = append(keyRunes, r)
			if r, ok = s.Next(); !ok {
				break
			}
		}

		// Skip any whitespace if we're not at the = yet
		if r != '=' {
			r, ok = s.SkipSpaces()
		}

		// The current character should be =
		if r != '=' || !ok {
			return fmt.Errorf(`missing "=" after %q in connection info string"`, string(keyRunes))
		}

		// Skip any whitespace after the =
		if r, ok = s.SkipSpaces(); !ok {
			// If we reach the end here, the last value is just an empty string as per libpq.
			o[string(keyRunes)] = ""
			break
		}

		if r != '\'' {
			for !unicode.IsSpace(r) {
				if r == '\\' {
					if r, ok = s.Next(); !ok {
						return fmt.Errorf(`missing character after backslash`)
					}
				}
				valRunes = append(valRunes, r)

				if r, ok = s.Next(); !ok {
					break
				}
			}
		} else {
		quote:
			for {
				if r, ok = s.Next(); !ok {
					return fmt.Errorf(`unterminated quoted string literal in connection string`)
				}
				switch r {
				case '\'':
					break quote
				case '\\':
					r, _ = s.Next()
					fallthrough
				default:
					valRunes = append(valRunes, r)
				}
			}
		}

		o[string(keyRunes)] = string(valRunes)
	}

	return nil
}

// newScanner returns a new scanner initialized with the option string s.
func (ext *postgresExtension) newScanner(s string) *scanner {
	return &scanner{[]rune(s), 0}
}

// Next returns the next rune.
// It returns 0, false if the end of the text has been reached.
func (s *scanner) Next() (rune, bool) {
	if s.i >= len(s.s) {
		return 0, false
	}
	r := s.s[s.i]
	s.i++
	return r, true
}

// SkipSpaces returns the next non-whitespace rune.
// It returns 0, false if the end of the text has been reached.
func (s *scanner) SkipSpaces() (rune, bool) {
	r, ok := s.Next()
	for unicode.IsSpace(r) && ok {
		r, ok = s.Next()
	}
	return r, ok
}
