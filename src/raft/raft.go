package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)
import "labrpc"

// import "bytes"
// import "labgob"



//
// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in Lab 3 you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh; at that point you can add fields to
// ApplyMsg, but set CommandValid to false for these other uses.
//
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int
}

type LogEntry struct {
	Term         int
	Command      interface{}
}

type ServeType int32
const (
	FOLLOWER          ServeType = 1
	CANDIDATE         ServeType = 2
	LEADER            ServeType = 3
)

//
// A Go object implementing a single Raft peer.
//
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	//
	//	// Your data here (2A, 2B, 2C).
	//	// Look at the paper's Figure 2 for a description of what
	//	// state a Raft server must maintain.
	//
	//
	//
	//	//perisit state
	currentTerm    int
	votedFor       int
	log            []LogEntry

	// all server state
	commitIndex  int
	lastApplied  int
	//leader, follower, candidate
	state        ServeType

	// leader state
	nextIndex    []int
	matchIndex   []int

	applyCh chan ApplyMsg
	beatCh chan int
	timer   *time.Timer

}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {

	var term int
	var isleader bool
	// Your code here (2A).
	rf.mu.Lock()
	term = rf.currentTerm
	isleader = rf.state == LEADER
	rf.mu.Unlock()
	return term, isleader
}


//
// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
//
func (rf *Raft) persist() {
	// Your code here (2C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// data := w.Bytes()
	// rf.persister.SaveRaftState(data)
}


//
// restore previously persisted state.
//
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (2C).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := labgob.NewDecoder(r)
	// var xxx
	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }
}



//
// example RequestVote RPC arguments structure.
// field names must start with capital letters!
//
type RequestVoteArgs struct {
	// Your data here (2A, 2B).
	Term  int
	CandidateId  int
	LastLogIndex   int
	LastLogTerm  int
}

//
// example RequestVote RPC reply structure.
// field names must start with capital letters!
//
type RequestVoteReply struct {
	// Your data here (2A).
	Term  int
	VoteGranted bool
}

//
// example RequestVote RPC handler.
//
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (2A, 2B).

	rf.mu.Lock()
	if args.Term < rf.currentTerm {
		reply.Term = rf.currentTerm
		reply.VoteGranted = false
	}else if args.Term > rf.currentTerm{
		reply.Term = args.Term
		reply.VoteGranted = true
		rf.currentTerm = args.Term
		rf.votedFor = args.CandidateId
		rf.beatCh <- args.CandidateId
	}else {
		if rf.votedFor == -1 || rf.votedFor == args.CandidateId {
			reply.Term = args.Term
			reply.VoteGranted = true
		}
	}

	rf.mu.Unlock()
}

//
// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
//
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

func (rf *Raft) sendAppendEntries(server int, args *AppendArgs, reply *AppendReplay) bool {
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	return ok
}



//
// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
//
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := true

	// Your code here (2B).
	rf.mu.Lock()
	isLeader = rf.state == LEADER
	term = rf.currentTerm
	index = len(rf.log)
	if isLeader {
		rf.log = append(rf.log, LogEntry{term, command})
	}
	rf.mu.Unlock()

	return index, term, isLeader
}

//
// the tester calls Kill() when a Raft instance won't
// be needed again. you are not required to do anything
// in Kill(), but it might be convenient to (for example)
// turn off debug output from this instance.
//
func (rf *Raft) Kill() {
	// Your code here, if desired.
}

type AppendArgs struct {
	// Your data here (2A, 2B).
	Term  int
	LeaderId int
	PrevLogIndex int
	PrevLogTerm int
	Entries  []LogEntry
	LeaderCommit  int
}

//
// example RequestVote RPC reply structure.
// field names must start with capital letters!
//
type AppendReplay struct {
	// Your data here (2A).
	Term   int
	Success  bool
}

func (rf *Raft) AppendEntries(args *AppendArgs, reply *AppendReplay) {
	// Your code here (2A, 2B).
	rf.mu.Lock()
	if args.Term > rf.currentTerm {
		rf.currentTerm = args.Term
	}
	rf.mu.Unlock()
	rf.beatCh <- args.LeaderId
}

func (rf *Raft) send_entries()  {
	args := AppendArgs{}
	args.Term = rf.currentTerm
	args.LeaderId = rf.me
	replay := AppendReplay{}
	for i := 0; i < len(rf.peers); i++ {
		server := i
		if server != rf.me {
			go func() {
				rf.sendAppendEntries(server, &args, &replay)
			}()
		}
	}
}

func (rf *Raft) run_leader() {
	rf.send_entries()
	timer := time.NewTimer(200 * time.Millisecond)

	for{
		select {
		case <- timer.C:
			rf.send_entries()
			timer.Reset(200 * time.Millisecond)
		case <- rf.beatCh:
			rf.mu.Lock()
			rf.state = FOLLOWER
			rf.mu.Unlock()
			return
		}
	}
}

func (rf *Raft) run_follower() {
	rf.resetTimer()
	for{
		select {
		case <- rf.timer.C:
			fmt.Println(rf.me, "in follower timeout")
			rf.mu.Lock()
			rf.state = CANDIDATE
			rf.mu.Unlock()
			return
		case <- rf.beatCh:
			fmt.Println(rf.me, "in follower receive leader")
			rf.resetTimer()
		}
	}

}

func (rf *Raft) run_candidate() {
	// Your code here, if desired.
	rf.mu.Lock()
	rf.currentTerm += 1
	countVote := 1
	rf.mu.Unlock()

	rf.resetTimer()

	// msg=1: 选举成功，成为leader
	// msg=2: 收到新leader信息，转为follower
	// msg=3: 选举超时，重新开始选举

	msgCh := make(chan int)
	stopCh := make(chan int)

	for i := 0; i < len(rf.peers); i++{
		server := i
		if server != rf.me {
			go func() {
				requestVoteArgs := RequestVoteArgs{}
				rf.mu.Lock()
				requestVoteArgs.Term = rf.currentTerm
				requestVoteArgs.CandidateId = rf.me
				rf.mu.Unlock()
				//requestVoteArgs.lastLogIndex = len(rf.log)
				//requestVoteArgs.lastLogTerm = rf.log[requestVoteArgs.lastLogIndex-1].term
				requestVoteReply := RequestVoteReply{}
				ok := rf.sendRequestVote(server, &requestVoteArgs, &requestVoteReply)
				if ok {
					rf.mu.Lock()
					if requestVoteReply.VoteGranted {
						countVote += 1
						if countVote > len(rf.peers) / 2{
							select {
							case <-stopCh:
							case msgCh <- 1:
							}
						}

					} else if requestVoteReply.Term > rf.currentTerm {
						rf.currentTerm = requestVoteReply.Term
						select {
						case <-stopCh:
						case msgCh <- 2:
						}
					}
					rf.mu.Unlock()
				}
			}()
		}
	}

	select {
	case cmd := <- msgCh:
		rf.mu.Lock()
		switch cmd {
		case 1:
			rf.state = LEADER
		case 2:
			rf.state = FOLLOWER
		case 3:
		}
		rf.mu.Unlock()
	case <- rf.timer.C:
	case <- rf.beatCh:
		rf.mu.Lock()
		rf.state = FOLLOWER
		rf.mu.Unlock()
	}
	close(stopCh)
}

func (rf *Raft) resetTimer() {
	rf.timer.Reset(rand_interal())
}

func (rf *Raft) run() {
	for {
		rf.mu.Lock()
		state := rf.state
		rf.mu.Unlock()

		// apply command
		go func() {

		}()

		switch state {
		case FOLLOWER:
			fmt.Println(rf.me, "become follower")
			rf.run_follower()
		case CANDIDATE:
			fmt.Println(rf.me, "become candidate")
			rf.run_candidate()
		case LEADER:
			fmt.Println(rf.me, "become leader")
			rf.run_leader()
		}

	}
}

//
// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
//
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me
	rf.applyCh = applyCh

	// Your initialization code here (2A, 2B, 2C).
	rf.currentTerm = 0
	rf.votedFor = -1

	rf.log = make([]LogEntry, 1)

	rf.commitIndex = -1
	rf.lastApplied = -1

	rf.nextIndex = make([]int, len(peers))
	rf.matchIndex = make([]int, len(peers))

	rf.beatCh = make(chan int)

	rf.state = FOLLOWER

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	rf.timer = time.NewTimer(rand_interal())
	go rf.run()

	return rf
}

func rand_interal()  time.Duration {
	var interval int = 400 + rand.Intn(200)
	return time.Duration(interval) * time.Millisecond
}
