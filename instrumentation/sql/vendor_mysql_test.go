package sql

import "testing"

func TestMySqlDSNParser(t *testing.T) {
	ext := &mysqlExtension{}
	dConf := &driverConfiguration{}

	ext.ProcessConnectionString("testUser:testPassword@tcp(localhost:3306)/dbname", dConf)

	if dConf.user != "testUser" {
		t.Fatal("User invalid.")
	}
	if dConf.port != "3306" {
		t.Fatal("Port invalid.")
	}
	if dConf.instance != "dbname" {
		t.Fatal("Instance invalid.")
	}
	if dConf.host != "localhost" {
		t.Fatal("Host invalid.")
	}
	if dConf.connString != "testUser:******@tcp(localhost:3306)/dbname" {
		t.Fatalf("Connection string invalid, expected: %s, actual: %s",
			"testUser:******@tcp(localhost:3306)/dbname",
			dConf.connString)
	}
}
