# How to use

- `make run`: run program and output raw timing data to startup.log. Assumes
  - You're logged into a cluster and can create DevWorkspaces
  - Default routingClass is valid for cluster
  - Namespace timing-test already exists
- `make cleanup`: delete any leftover devworkspaces created by the program
- `make process`: output CSV-formatted table of timing data; depends on `jq`

Constants 
```go	
maxContainers = 10
iterations    = 10
```
can be changed to control max number of containers, and how many times the DevWorkspace is started for each container count.