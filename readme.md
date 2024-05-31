## Simple Go Echo Demo using QUIC

There is a single binary that is used to run both the client and the server.

### Note
Dr. Mitchell said
"I just got off a help call with another student using Go, he wasnt having any issues with the
code on Windows.  Go is very portable, so I will relax the unix requirement for projects done in
go"

So, Tux has a different version of Go than the start app (https://github.com/ArchitectingSoftware/CS544-Class-Demo-Files/tree/main/quic/go)
called for, but per Dr. Mitchell's message he is relaxing requirements for Go. 

### Extra Credit
- Includes
  - Demo Skills: YouTube video here https://youtu.be/TC5Izxmwq70
  - Implementation Robustness: implemented all of the agreed-upon functionality from Proj-Part3a: Implementation Proposal
  - Concurrent or Asynchronous Server: Uses concurrent/asynch server(s)
  - Using a Systems Programming Language: Used Go (new programming language for me)
  - Providing a summary of how focusing on the implementation increased learning and resulted in needing to update your protocol specification, **see bottom of README for this summary**
  - Working with a cloud-based git-based system: Project is here https://github.com/mattjg908/cs544_project3/
  - Have the client program dynamically find the server: Sort of, you do have to pass the second server's port number but the rest of it is automatic
  - Include with your code automated testing code: N/A
  - Active help of your fellow classmates: N/A
  - Design Excellence: Determined by grader :)

For testing:
### Single Server Testing
- in a terminal window, start the `server`: `go run cmd/echo/echo.go -server`
- in another terminal window, connect to the `server` with a client (nickname is `foo`): `go run cmd/echo/echo.go -client -mtype=connect -data="foo|password123"`
- in another terminal window, connect to the `server` with a second client (nickname is `bar`) : `go run cmd/echo/echo.go -client -mtype=connect -data="bar|password123"`
- on the `foo` client, list all connected clients. You should see `foo,bar`: `list`
- on the `foo` client, send a message to `bar`. This is done by typing `recipient|some message` like: `bar|hello world`
- on the `bar` client, you will see `foo: hello world`
- now send a message from the `bar` client to the `foo` client: `foo|hi back!`
- on the `foo` client, you will see `bar: hi back!`
- on the `foo` client, mark yourself as away: `away`
- on the `bar` client, send a message to `foo`: `foo|are you there?`
- on the `bar` client, you will see an auto-response message: `I am away`
- on the `foo` client, toggle being away: `away`
- on the `bar` client, send another message to `foo`: `foo|welcome back`
- on the `bar` client, note there is no auto-reply

### Multiple Server Testing
- `server`:
  - to start a server:
    - `go run cmd/echo/echo.go -server`
  - to start a second server, use a different port number (and/or `server-ip`):
    - `go run cmd/echo/echo.go -server -port=4243`
- `client`:
  - to connect:
    - `go run cmd/echo/echo.go -client -mtype=connect -data="foo|password123"`
  - to connect to a different server, pass that server's port number (and/or server-ip):
    - `go run cmd/echo/echo.go -client -mtype=connect -data="bar|password123" -port=4243`
  - follow all the same steps above as you did for the single server. You will see that the users being on different servers is transparent. Users can message each, list users, and mark themselves as `away` exactly the same way the did with a single `server`

Solution derived from the excellent work of:
- the `quic-go` team based on the example: https://github.com/quic-go/quic-go/tree/master/example/echo
- the Go exmaple from Dr. Mitchell: https://github.com/ArchitectingSoftware/CS544-Class-Demo-Files/tree/main/quic/go

## Extra Credit Summary (as mentioned above):
Focusing on implementation increased learning and resulted in needing to update my protocol specification in several ways. The biggest was that prior to implementation, I had a
mental model whereby all my PDUs were different. I received feedback on Part 2 of this assignment that noted my PDU design could have been simpler. It was not until I actually started
the implementation that I truly grasped that the PDU is actually very generic but can handle many different kinds of messages efficiently. Another way focusing on implementation helped
with learning was that I have never used Go before, I really enjoyed it. Working with Go Routines, and thinking about how the same code can run multiple clients/servers, some of which
is ran inside a Go routine for more concurrency also made me stretch my brain a bit. I was similarly surprised that once the PDU and Echo modules had their initial desing in place, each
time I needed to add functionlaity I really just needed to update the Client and Server modules. I was surprised how nicely encapsulated all this was. Also, I appreciated how easy Go
makes doing stuff like this. I don't have any professional experience with C, but if Go wasn't here I would have chosen C and I know this would have been a lot harder in C. I also got
my hands dirty with how Streams work with sending the PDUs which really helped me to grasp wwhat was going on. Also, implementing this chat app was a good way to work with the 
client-server pattern, which in turn helped me think about how to send and receive packets of data, and how to create those packets.
