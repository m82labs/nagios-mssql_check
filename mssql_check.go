package main

import (
	"database/sql"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	_ "github.com/denisenkom/go-mssqldb"
)

func main() {
	var (
		userid     = flag.String("U", "sa", "User name to connect with")
		password   = flag.String("P", "password!", "User password")
		server     = flag.String("h", "localhost", "server_name[\\instance_name]")
		database   = flag.String("d", "master", "Database name to connect to")
		filepath   = flag.String("s", "test.sql", "File path to SQL script file")
		argument   = flag.String("a", "", "SQLCMD-style arguments: 'Key1=Value1;Key2=Value2'")
		integrated = flag.Bool("i", false, "Enable integrated (Windows) authentication")
		get_timing = flag.Bool("t", false, "Returns timing data ONLY for the query executed, normal query output is omitted.")
	)
	flag.Parse()

	// Connection string
	var dsn string
	if *integrated {
		dsn = "server=" + *server + ";database=" + *database
	} else {
		dsn = "server=" + *server + ";user id=" + *userid + ";password=" + *password + ";database=" + *database
	}
	db, err := sql.Open("mssql", dsn)
	if err != nil {
		fmt.Println("Cannot connect: ", err.Error())
		os.Exit(3)
	}

	err = db.Ping()
	if err != nil {
		fmt.Println("Cannot connect: ", err.Error())
		os.Exit(3)
	}
	defer db.Close()

	// Read the script file
	script, err := ioutil.ReadFile(*filepath)
	if err != nil {
		fmt.Println("Cannot open script file: ", err.Error())
		os.Exit(3)
	}

	cmd := string(script)

	if *argument != "" {
		for _, arg := range strings.Split(*argument, ",") {
			currarg := strings.Split(arg, ":")
			if cap(currarg) == 2 {
				cmd = strings.Replace(cmd, "$("+currarg[0]+")", currarg[1], -1)
			} else {
				fmt.Println("Error parsing argurments. Key=Value pair not found.")
				os.Exit(3)
			}
		}
	}

	// Capture timings
	var start time.Time
	var dur int64
	if *get_timing {
		start = time.Now()
	}

	// Execute the script
	rows, err := db.Query(cmd)
	if err != nil {
		fmt.Println("Failed to execute script: ", err.Error())
		os.Exit(3)
	}
	defer rows.Close()

	// Calculate total time
	if *get_timing {
		dur = time.Since(start).Nanoseconds() / 1000000
	}

	// Read three result sets
	// Servier Status
	var service_status string
	var current_hr string

	for rows.Next() {
		err = rows.Scan(&current_hr)
		if err != nil {
			fmt.Println("Failed to parse results: ", err)
			os.Exit(3)
		}
		service_status += current_hr + "\n"
	}

	// Performance data
	if rows.NextResultSet() {
		if *get_timing {
			service_status = fmt.Sprintf("Response Time: %dms", dur)
			service_status += fmt.Sprintf("|es=1,instance_latency_ms=%d", dur)
			for rows.Next() {
				// Do nothing here
			}
		} else {
			var metric string
			var value int64
			service_status += "|"
			for rows.Next() {
				err = rows.Scan(&metric, &value)
				if err != nil {
					fmt.Println("Failed to gather performance data: ", err)
					os.Exit(3)
				}
				service_status += fmt.Sprintf("%s=%d", metric, value)

			}
		}
	} else {
		fmt.Println("No performance data found.")
		os.Exit(3)
	}

	// Exit Code
	var exitcode int = 3 //default to UNKNOWN
	if rows.NextResultSet() {
		if *get_timing {
			exitcode = 0
		} else {
			for rows.Next() {
				err = rows.Scan(&exitcode)
				if err != nil {
					fmt.Println(err)
					continue
				}
			}
		}
	} else {
		fmt.Println("No exit code found.")
		os.Exit(3)
	}

	// Output service status information
	fmt.Println(service_status)

	// Exit with the exit code
	os.Exit(exitcode)
}
