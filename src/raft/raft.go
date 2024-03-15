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
	//	"bytes"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	//	"6.5840/labgob"
	"6.5840/labrpc"
)

// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in part 3D you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh, but set CommandValid to false for these
// other uses.
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int

	// For 3D:
	SnapshotValid bool
	Snapshot      []byte
	SnapshotTerm  int
	SnapshotIndex int
}

type Role int

const (
	Follower Role = iota
	Candidate
	Leader
)

// the inf of log entry
type LogEntry struct {
	Term    int
	Command interface{}
}

// A Go object implementing a single Raft peer.
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()

	// Your data here (3A, 3B, 3C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.

	// Persistent state on all servers
	currentTerm int
	votedFor    int
	logs        []LogEntry

	// Volatile state on all servers
	commitIndex int
	lastApplied int

	// Volatile state on leaders
	nextIndex  []int
	matchIndex []int

	role       Role
	heartChan  chan struct{}
	lastUpdate time.Time
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {

	var term int
	var isleader bool
	// Your code here (3A).

	rf.mu.Lock() // 先加锁，之后用-race检测
	term = rf.currentTerm
	isleader = rf.role == Leader
	rf.mu.Unlock()

	return term, isleader
}

// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
// before you've implemented snapshots, you should pass nil as the
// second argument to persister.Save().
// after you've implemented snapshots, pass the current snapshot
// (or nil if there's not yet a snapshot).
func (rf *Raft) persist() {
	// Your code here (3C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// raftstate := w.Bytes()
	// rf.persister.Save(raftstate, nil)
}

// restore previously persisted state.
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (3C).
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

// the service says it has created a snapshot that has
// all info up to and including index. this means the
// service no longer needs the log through (and including)
// that index. Raft should now trim its log as much as possible.
func (rf *Raft) Snapshot(index int, snapshot []byte) {
	// Your code here (3D).

}

func (rf *Raft) isLogUpToDate(lastLogTerm, lastLogIndex int) bool {
	if lastLogTerm == rf.logs[len(rf.logs)-1].Term {
		return lastLogIndex >= len(rf.logs)-1
	}

	return lastLogTerm > rf.logs[len(rf.logs)-1].Term
}

// example RequestVote RPC arguments structure.
// field names must start with capital letters!
type RequestVoteArgs struct {
	// Your data here (3A, 3B).
	Term         int
	CandidateId  int
	LastLogIndex int
	LastLogTerm  int
}

// example RequestVote RPC reply structure.
// field names must start with capital letters!
type RequestVoteReply struct {
	// Your data here (3A).
	Term        int
	VoteGranted bool
}

// RequestVote RPC handler.
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (3A, 3B).
	rf.mu.Lock()
	defer rf.mu.Unlock()

	// 1. term < currentTerm: reply false
	// 2. term == currentTerm and have voted: reply false
	// 3. term > currentTerm: convert to follower, reset rf
	// 4. term == currentTerm and not vote: compare log
	// 5. compare lastLogIndex and lastLogTerm

	if args.Term < rf.currentTerm {
		reply.Term, reply.VoteGranted = rf.currentTerm, false
		return
	} else if args.Term == rf.currentTerm && rf.votedFor != -1 && rf.votedFor != args.CandidateId {
		reply.Term, reply.VoteGranted = rf.currentTerm, false
		return
	}

	if args.Term > rf.currentTerm {
		if rf.role == Leader {
			rf.stepDown()
		}
		rf.role = Follower // TODO: 把 role 互相转变的过程封装起来，convert默认在加锁情况下转变
		rf.votedFor = -1
	}

	// 如果符合条件的candidate的日志不是最新的，不投票给他，同时重置votedFor
	if !rf.isLogUpToDate(args.LastLogTerm, args.LastLogIndex) {
		reply.Term, reply.VoteGranted = rf.currentTerm, false
		rf.votedFor = -1
		return
	}

	reply.Term, reply.VoteGranted = rf.currentTerm, true

	rf.votedFor = args.CandidateId
	rf.lastUpdate = time.Now()
	DPrintf("{Server %v} Term %v Vote for Server %v\n", rf.me, rf.currentTerm, rf.votedFor)
}

func (rf *Raft) genRequestVoteArgs() RequestVoteArgs {
	// rf.mu.Lock()
	// defer rf.mu.Unlock()

	return RequestVoteArgs{
		Term:         rf.currentTerm,
		CandidateId:  rf.me,
		LastLogIndex: len(rf.logs) - 1,
		LastLogTerm:  rf.logs[len(rf.logs)-1].Term,
	}
}

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
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

type AppendEntriesArgs struct {
	Term         int
	LeaderId     int // followers can redirect client
	PrevLogIndex int
	PrevLogTerm  int
	Entries      []LogEntry
	LeaderCommit int // leader's commitIndex
}

type AppendEntriesReply struct {
	Term    int
	Success bool // true if follower contained entry matching prevLogIndex and prevLogTerm
}

func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	// if AppendEntries RPC received from now leader: convert to follower
	if args.Term >= rf.currentTerm {
		// 如果任期号大于或等于当前节点的任期号，需要转换为follower
		rf.role = Follower // TODO
		rf.currentTerm = args.Term
		rf.lastUpdate = time.Now()

		reply.Term = args.Term
		reply.Success = true
	} else {
		reply.Term = rf.currentTerm
		reply.Success = false
	}
}

func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	return ok
}

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
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := true

	// Your code here (3B).

	return index, term, isLeader
}

func (rf *Raft) startElection() {
	rf.role = Candidate
	rf.currentTerm++
	rf.votedFor = rf.me
	DPrintf("{Server %v} Term %v Start Election\n", rf.me, rf.currentTerm)

	args := rf.genRequestVoteArgs()

	// send RequestVote RPCs to all other servers
	voteCount := 1

	for i := range rf.peers {
		if i != rf.me {
			// 将i作为闭包传递给goroutine，因为直接在goroutine中调用i，可能会调用已经修改过的i
			DPrintf("{Server %v} Term %v Send RequestVoteArgs to server %v\n",
				rf.me, rf.currentTerm, i)
			go func(server int) {
				reply := RequestVoteReply{}
				if rf.sendRequestVote(server, &args, &reply) {
					rf.mu.Lock()
					defer rf.mu.Unlock()
					if rf.role == Candidate && reply.VoteGranted {
						voteCount++
						// if votes received from majority of servers: become leader
						if voteCount > len(rf.peers)/2 {
							rf.role = Leader
							DPrintf("{Server %v} Term %v Success become leader.\n",
								rf.me, rf.currentTerm)
							rf.becomeLeader()
							return
						}
					} else if reply.Term > rf.currentTerm {
						DPrintf("{Server %v} Old Term %v discovers a new term %v, convert to follower\n",
							rf.me, rf.currentTerm, reply.Term)
						rf.currentTerm = reply.Term
						rf.role = Follower // TODO
					}
				}

			}(i)
		}
	}

}

// 选举成功时，调用此函数
func (rf *Raft) becomeLeader() {
	// rf.mu.Lock()
	rf.heartChan = make(chan struct{})
	// rf.mu.Unlock()

	go rf.startHeartbeat()
	DPrintf("{Server %v} Term %v start heartbeat\n", rf.me, rf.currentTerm)
}

// 当Leader转变为Candidate/Follower
func (rf *Raft) stepDown() {
	// rf.mu.Lock()
	close(rf.heartChan) // 停止发送heartbeat
	// rf.mu.Unlock()
}

// Upon election: send initial empty AppendEntries RPCs(heartbeat)
// to each server; repeat during idle periods to prevent
// election timeouts (#5.2)
func (rf *Raft) startHeartbeat() {
	// 每秒钟发送8次，测试程序要求leader每秒发送的心跳包次数不超过10次
	interval := time.Second / 8

	go func() {
		for {
			select {
			case <-rf.heartChan:
				return
			default:
				term, isLeader := rf.GetState()

				// 当服务器是Leader时，发送心跳包
				if isLeader {
					args := AppendEntriesArgs{Term: term, LeaderId: rf.me}

					// 并行向所有Follower send AppendEntries
					for i := range rf.peers {
						if i != rf.me {
							go func(server int) {
								reply := AppendEntriesReply{}
								rf.sendAppendEntries(server, &args, &reply)
							}(i)
						}
					}
				} else {
					return
				}
				// wait for next send AppendEntries
				time.Sleep(interval)
			}
		}
	}()
}

// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

// election timeout
func (rf *Raft) ticker() {
	for rf.killed() == false {
		// Your code here (3A)
		// Check if a leader election should be started.

		_, isLeader := rf.GetState()
		// DPrintf("{Server %v} Term %v check election timeout\n", rf.me, term)

		rf.mu.Lock()

		// elecion timeout elapses: start new elections
		if !isLeader && time.Since(rf.lastUpdate) > 500*time.Millisecond {
			rf.startElection()
		}
		rf.mu.Unlock()

		// pause for a random amount of time between 50 and 350
		// milliseconds.
		// ms := 50 + (rand.Int63() % 300)
		ms := 600 + (rand.Int63() % 300)
		time.Sleep(time.Duration(ms) * time.Millisecond)
	}
}

// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me
	rf.dead = 0

	// Your initialization code here (3A, 3B, 3C).
	rf.currentTerm = 0
	rf.logs = []LogEntry{{Term: 0}}
	rf.role = Follower

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	// start ticker goroutine to start elections
	go rf.ticker()

	return rf
}
