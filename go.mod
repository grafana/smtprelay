module github.com/decke/smtprelay

require (
	github.com/chrj/smtpd v0.2.0
	github.com/google/uuid v1.2.0
	github.com/prometheus/client_golang v1.17.0
	github.com/sirupsen/logrus v1.7.0
	github.com/vharitonsky/iniflags v0.0.0-20180513140207-a33cd0b5f3de
	golang.org/x/crypto v0.0.0-20210921155107-089bfa567519
)

// remove this once this PR https://github.com/chrj/smtpd/pull/12 is merge
replace github.com/chrj/smtpd => github.com/ebilling/smtpd v0.3.1-0.20210929184440-24056bf10d0e

go 1.13
