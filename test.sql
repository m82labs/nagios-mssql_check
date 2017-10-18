SELECT TOP (1) name AS HR  FROM sys.databases
UNION ALL
SELECT name AS HR FROM sys.databases

SELECT 'Metric' AS Metric, 10 AS Value

SELECT 3 AS exit_code
