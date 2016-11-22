// Go IMAP roadrunner
// Simple IMAP benchmarking tool

// Apache License
// Author: Ben Duncan, Atmail CTO
// (C) Atmail Pty Ltd 

package main

import(
	"log"
	"fmt"
	"github.com/mxk/go-imap/imap"
	"errors"
	"bytes"
	"time"
	"flag"
	"os"
	"strings"
)


func main()	{
	
	start := time.Now()

	// Required CLI flags to connect to the remote IMAP server
	user := flag.String("user", "", "Username to authenticate (required)")
	pass := flag.String("pass", "", "Password to authenticate (required)")
	server := flag.String("server", "", "Remote IMAP server (required)")

	// Optional flags
	folder := flag.String("folder", "Inbox", "Folder to select")
	cycle := flag.Int("cycle", 3, "Number of times to cycle")
	csv := flag.Bool("csv", false, "Flag for CSV output")

	flag.Parse()

	// Validate the user CLI input
	if *user == ""  {
		fmt.Println("User missing from arguments")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Require a password
	if *pass == ""  {
		fmt.Println("Password missing from arguments")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Require a server to be specified
	if *server == ""  {
		fmt.Println("Server missing from arguments")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// If not exporting in CSV format, display our header banner
	if(*csv == false)	{
		fmt.Println("Go IMAP Roadrunner at your service")
	} else {
		fmt.Println("\"Cycle\", \"Message ID\", \"Message in Bytes\", \"Execution time in secs\"")
	}

	// Run the IMAP sequence X times, useful for benchmarking the same request multiple times.
	for i := 1; i <= *cycle; i++	{

		if(*csv == false)	{
			fmt.Println("Launch cycle ", i)
		}

		// Run the test
		BenchmarkIMAP(*folder, *user, *pass, *server, *csv, i)
	}

	// Output the total run-time, if we are not exporting to CSV
	if(*csv == false)	{
		elapsed := time.Since(start)
		fmt.Println("Total run time => ", elapsed)
	}

}


func BenchmarkIMAP(folder, user, pass, server string, csv bool, cycle int)	{

	// First, connect to the specified IMAP server
	c, total_msgs, err := ConnectIMAP(folder, user, pass, server)

	// Can we connect?
	if(err != nil)	{
		log.Fatal("Could not connect to IMAP: ", err)
	}

	// Require messages to exist in the folder, exit otherwise.
	if(c == nil)	{
		log.Fatal("Folder count for ", folder, "is empty. Please check the mailbox contains messages")
	}

	// If not using CSV, export the number of messages contained in the folder.
	if(csv == false)	{
		fmt.Println("Server responded with =>", total_msgs, "total messages in", folder)
	}

	// Loop through each message, request the entire message and time the response
	for i := 1; i <= int(total_msgs); i++	{
		set, _ := imap.NewSeqSet("")
		set.AddNum(uint32(i))
		FetchMail(c, set, csv, cycle)
	}

	// Logout immediately once complete
	c.Logout(0)

}

func ConnectIMAP(folder, user, passwd, server string) (client *imap.Client, total uint32, err error)	{

	// Connect to the specified server, port 143
	// TODO: Add TLS support and IMAP(s)
	s := []string{server, ":143"}
	hostname := strings.Join(s, "")

	c, err := imap.Dial(hostname)

	// If the host does not connect, return an error
	if err != nil {
		return c, 0, err
	}

	if c.State() == imap.Login {
		c.Login(user, passwd)
	}

	// Confirm the username/password combination is successful
	if c.State() != imap.Auth	{
		log.Println("Authentication failed")
		return c, 0, errors.New("Cannot authenticate: " + user)
	}

	// Select the specified folder, used for fetch requests next
	c.Select(folder, true)

	return c, c.Mailbox.Messages, nil
}

func FetchMail(c *imap.Client, seq *imap.SeqSet, csv bool, cycle int ) (total uint32)	{

	start := time.Now()

	// Specify the headers required
	// TODO: Specify these on the CLI as an optional argument
	items := []string{
		"INTERNALDATE",
		"FLAGS",
		"RFC822.SIZE",
		"BODY.PEEK[]",
	}

	// TODO: Wait, sync, if no wait, polls until buffer used?
	cmd, err := imap.Wait(c.Fetch(seq, items...))

	// Return if there is an error from the remote IMAP (e.g message no longer exists)
	if err != nil	{
		log.Println("Message", seq, "could not be retrieved:( ", err, ")")
		return 0
	}

	// Read the raw IMAP response into a buffer
	rsp := cmd.Data[0]
	buf := new(bytes.Buffer)

	for _, l := range rsp.Literals {
		l.WriteTo(buf)
	}

	Content := buf.String()
	size := len(Content)
	elapsed := time.Since(start)

	// Output the stats on the message ID, size and how long the request took.
	if(csv == true)	{
		fmt.Println(cycle, "," , seq, "," , size, ",", elapsed.Seconds())
	} else {
		fmt.Println("Message ID:", seq, ">> IMAP reply =>", size , "bytes", "( received in ", elapsed.Seconds(), ")")

	}

	// TODO: Return the stats above, let the parent function handle the output.
	return 0
}