package sql

import "testing"

func TestPostgresDSNParser(t *testing.T) {
	ext := &postgresExtension{}
	dConf := &driverConfiguration{}

	ext.ProcessConnectionString("host=localhost port=5432 user=testUser password=testPassword dbname=dbname", dConf)

	if dConf.user != "testUser" {
		t.Fatal("User invalid.")
	}
	if dConf.port != "5432" {
		t.Fatal("Port invalid.")
	}
	if dConf.instance != "dbname" {
		t.Fatal("Instance invalid.")
	}
	if dConf.host != "localhost" {
		t.Fatal("Host invalid.")
	}
	if dConf.connString != "host=localhost port=5432 user=testUser password=****** dbname=dbname" {
		t.Fatal("Connection string invalid.")
	}
}