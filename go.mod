module github.com/decke/smtprelay

require (
	github.com/chrj/smtpd v0.2.0
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/uuid v1.2.0
	github.com/kr/pretty v0.2.0 // indirect
	github.com/prometheus/client_golang v1.9.0
	github.com/sirupsen/logrus v1.9.3
	github.com/vharitonsky/iniflags v0.0.0-20180513140207-a33cd0b5f3de
	golang.org/x/crypto v0.0.0-20210921155107-089bfa567519
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200605160147-a5ece683394c // indirect
)

// remove this once this PR https://github.com/chrj/smtpd/pull/12 is merge
replace github.com/chrj/smtpd => github.com/ebilling/smtpd v0.3.1-0.20210929184440-24056bf10d0e

go 1.13
