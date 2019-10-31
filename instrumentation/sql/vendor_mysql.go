package sql

import (
	"net"
	"strings"
)

type mysqlExtension struct{}

func init() {
	vendorExtensions = append(vendorExtensions, &mysqlExtension{})
}

// Gets if the extension is compatible with the component name
func (ext *mysqlExtension) IsCompatible(componentName string) bool {
	return componentName == "mysql.MySQLDriver" ||
		componentName == "godrv.Driver" ||
		componentName == "driver.driver"
}

// Complete the missing driver data from the connection string
func (ext *mysqlExtension) ProcessConnectionString(connectionString string, configuration *driverConfiguration) {
	configuration.peerService = "mysql"

	dsn := *ext.parseDSN(connectionString)
	configuration.user = dsn["User"]
	configuration.port = dsn["Port"]
	configuration.instance = dsn["DBName"]
	configuration.host = dsn["Host"]
	configuration.connString = strings.ReplaceAll(connectionString, dsn["Passwd"], "******")
}

// ParseDSN parses the DSN string to a Config
func (ext *mysqlExtension) parseDSN(dsn string) *map[string]string {
	// New config with some default values
	tmpCfg := map[string]string{}

	// [user[:password]@][net[(addr)]]/dbname[?param1=value1&paramN=valueN]
	// Find the last '/' (since the password or the net addr might contain a '/')
	foundSlash := false
	for i := len(dsn) - 1; i >= 0; i-- {
		if dsn[i] == '/' {
			foundSlash = true
			var j, k int

			// left part is empty if i <= 0
			if i > 0 {
				// [username[:password]@][protocol[(address)]]
				// Find the last '@' in dsn[:i]
				for j = i; j >= 0; j-- {
					if dsn[j] == '@' {
						// username[:password]
						// Find the first ':' in dsn[:j]
						for k = 0; k < j; k++ {
							if dsn[k] == ':' {
								tmpCfg["Passwd"] = dsn[k+1 : j]
								break
							}
						}
						tmpCfg["User"] = dsn[:k]
						break
					}
				}

				// [protocol[(address)]]
				// Find the first '(' in dsn[j+1:i]
				for k = j + 1; k < i; k++ {
					if dsn[k] == '(' {
						// dsn[i-1] must be == ')' if an address is specified
						if dsn[i-1] != ')' {
							if strings.ContainsRune(dsn[k+1:i], ')') {
								return nil
							}
							return nil
						}
						tmpCfg["Addr"] = dsn[k+1 : i-1]
						break
					}
				}
				tmpCfg["Net"] = dsn[j+1 : k]
			}

			// dbname[?param1=value1&...&paramN=valueN]
			// Find the first '?' in dsn[i+1:]
			for j = i + 1; j < len(dsn); j++ {
				if dsn[j] == '?' {
					ext.parseDSNParams(&tmpCfg, dsn[j+1:])
					break
				}
			}
			tmpCfg["DBName"] = dsn[i+1 : j]
			break
		}
	}

	if !foundSlash && len(dsn) > 0 {
		return nil
	}
	ext.normalize(&tmpCfg)
	return &tmpCfg
}

// parseDSNParams parses the DSN "query string"
// Values must be url.QueryEscape'ed
func (ext *mysqlExtension) parseDSNParams(cfg *map[string]string, params string) {
	for _, v := range strings.Split(params, "&") {
		param := strings.SplitN(v, "=", 2)
		if len(param) != 2 {
			continue
		}
		(*cfg)[param[0]] = param[1]
	}
}

func (ext *mysqlExtension) normalize(cfg *map[string]string) {
	// Set default network if empty
	if (*cfg)["Net"] == "" {
		(*cfg)["Net"] = "tcp"
	}

	// Set default address if empty
	if (*cfg)["Addr"] == "" {
		switch (*cfg)["Net"] {
		case "tcp":
			(*cfg)["Addr"] = "127.0.0.1:3306"
		case "unix":
			(*cfg)["Addr"] = "/tmp/mysql.sock"
		}
	} else if (*cfg)["Net"] == "tcp" {
		(*cfg)["Addr"] = ext.ensureHavePort((*cfg)["Addr"])
	}

	if host, port, err := net.SplitHostPort((*cfg)["Addr"]); err == nil {
		(*cfg)["Host"] = host
		(*cfg)["Port"] = port
	}
}

func (ext *mysqlExtension) ensureHavePort(addr string) string {
	if _, _, err := net.SplitHostPort(addr); err != nil {
		return net.JoinHostPort(addr, "3306")
	}
	return addr
}
