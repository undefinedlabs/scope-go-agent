package sql

import (
	"fmt"
	"net"
	nurl "net/url"
	"sort"
	"strings"
)

func fillPostgresDriverData(name string, w *instrumentedDriver) {
	w.configuration.peerService = "postgresql"

	dsn := name
	if strings.HasPrefix(name, "postgres://") || strings.HasPrefix(name, "postgresql://") {
		if pDsn, err := postgresParseURL(name); err == nil {
			dsn = pDsn
		}
	}
	o := make(values)
	o["host"] = "localhost"
	o["port"] = "5432"
	_ = parseOpts(dsn, o)
	o["password"] = "******"

	if user, ok := o["user"]; ok {
		w.configuration.user = user
	}
	if port, ok := o["port"]; ok {
		w.configuration.port = port
	}
	if dbname, ok := o["dbname"]; ok {
		w.configuration.instance = dbname
	}
	if host, ok := o["host"]; ok {
		w.configuration.host = host
	}

	w.configuration.connString = fmt.Sprint(o)
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
func postgresParseURL(url string) (string, error) {
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
