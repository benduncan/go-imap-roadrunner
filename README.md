# Go RoadRunner

![Roadrunner Screenshot](./go-roadrunner.png)

###### Version 1.0 (beta) - Ben Duncan, Atmail Pty Ltd

### Introduction

RoadRunner is designed as a simple command-line tool for benchmarking an IMAP server. The tool will connect with a specified username/password and fetch mail for a specified folder while timing the result.

The tool is useful to benchmark IMAP servers, especially when switching mailbox formats, making changes to the backend mail-store via NFS, or when implementing object-storage features for user mailboxes.

RoadRunner will put the IMAP server through it's paces, so you as the sys-admin can relax and have peace of mind systems are fully *operational*.

### Dependencies

Roadrunner is designed for the least number of dependencies possible.

* [Go IMAP](https://github.com/mxk/go-imap/)  

### Compile

Compiling is easy

```
go get ./...
go build roadrunner.go
```

### Usage

The following arguments are required

```
  -csv
    	Flag for CSV output
  -cycle int
    	Number of times to cycle (default 3)
  -folder string
    	Folder to select (default "Inbox")
  -pass string
    	Password to authenticate (required)
  -server string
    	Remote IMAP server (required)
  -user string
    	Username to authenticate (required)
```

### Example output


```
./roadrunner -user ben -pass abc123 -server localhost

Go IMAP Roadrunner at your service
Launch cycle  1
Server responded with => 89 total messages in Inbox
Message ID: 1 >> IMAP reply => 12271465 bytes ( received in  0.5390176640000001 )
Message ID: 2 >> IMAP reply => 21972924 bytes ( received in  1.517693585 )
Message ID: 3 >> IMAP reply => 1721802 bytes ( received in  0.157302838 )

```

### Export to CSV

Need to export the format to CSV and graph the results? No problem, simply add the -csv flag

```
./roadrunner -user ben -pass abc132 -server localhost -csv > /tmp/imap-benchmark.csv

```

### Report sample

Once authenticated, RoadRunner will select the specified folder, and loop through all messages in the specified folder and fetch the raw message while timing the response.

```
"Cycle", "Message ID", "Message in Bytes", "Execution time in secs"
1 , 1 , 12271465 , 0.565702689
1 , 2 , 21972924 , 1.238866853
1 , 3 , 1721802 , 0.050218485
...
```
