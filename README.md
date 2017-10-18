# MSSQL Check
A SQL Server plugin for Nagios written in Go.

## Overview
A simple Nagios plugin to execute arbitrary SQL scripts against an instance.

### Features

- Support for SQLCMD style variable substitutions
- Support for "timing" mode to measure duration for any query (useful for canary style queries)
- Support for all standard plugin features:
  - Service Status
  - Detailed Service Status
  - Optional Performance Data
  - Service Status Code

## Nagios Configuration
In order to use this plugin, you will need to create a command definition, and a service definition.

Here is an example config that assumes the `$USER1$` macro points to your `libexec` directory, `$USER2$` points to the directory you plan to store your SQL scripts, and `$USER3$` and `$USER4$` are set to the SQL account you will be using for SQL authentication.

```
#MSSQL Query-Based Checks
define command{
    command_name mssql_check
    command_line $USER1$/mssql_check -h $HOSTNAME$ -s $USER2$/$ARG1$ $ARG2$
}
```

Using this example, your service checks would be written as follows:
```
define service {
...
    ; This command takes 1 argument:
    ;  - last_check (int): The timestamp from the last check in UNIX Epoch time (seconds since 01/01/1970)
    check_command mssql_check!CriticalErrorCheck.sql!-a "last_check:$LASTSERVICECHECK$"
...
}
```

This will pass the name of the SQL script to execute as `$ARG1$` and `-a "last_check:$LASTSERVICECHECK$"` as `$ARG2$`. As a general rule, I like to document SQLCMD variables within the service check definition, to avoid confusion. If you are using the `-t` option, you would add it to the end of the `check_command` line above.

## SQL Scripts
In order to use a SQL script file with this plugin, it must return three result sets:

1. Service status, human readable data: The first row will be included in outgoing alerts and be displayed on summary screen in Nagios. Any additional rows in the first result set will be displayed when viewing the details of a specific service check.
1. Performance data: For some checks you may want to include performance data of some kind that will be written to an external system or used to generate graphs. This result set should be in the form of `Key,Value`, each row being a separate key/value pair. If you are not going to return performance data, simply add `SELECT 'Key',NULL AS [Value]` as the second query in your script.
1. Exit code: Returns a single integer representing the service status (0 - OK, 1 - Warning, 2 - Critical, 3 - Unknown)

### Simple Example
```
// Service Status
SELECT TOP (1) name AS HR  FROM sys.databases
UNION ALL
SELECT name AS HR FROM sys.databases

// Performance Data
SELECT 'Metric' AS Metric, 10 AS Value

// Exit Code
SELECT 0 AS exit_code
```

### Variables
This plugin has full support for SQLCMD style variable substitution. 

For example:
```
DECLARE	@nagios_time INT = $(last_check)
DECLARE @start_date DATETIME2(3);

SET @start_date = DATEADD(second,@nagios_time,'01/01/1970')

SELECT * FROM table WHERE event_date >= @start_date
...
```

In this snippet we might want to check a table for events that occurred since the last check. To do this, we use a SQLCMD variable `$(last_check)`. Using the `-a` parameter of this plugin, you can pass in the name of the variable and the time of the last check (using a Nagios macro): 
```
-a "last_check:$LASTSERVICECHECK$"
```

## Installation
To install:

1. Install Go
1. Set your $GOPATH
1. `go get -d github.com/m82labs/nagios-mssql_check`
1. `cd $GOPATH/src/github.com/m82labs/nagios-mssql_check`
1. `go build -o mssql_check`
1. Copy the resulting executable into you Nagios `libexec` directory.
