package main

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rulestone/actors"
	"github.com/rulestone/api"
	"github.com/rulestone/condition"
	"github.com/rulestone/engine"
	"github.com/rulestone/types"
	"github.com/rulestone/utils"
	"io"
	"net"
	"os"
	"path/filepath"
	"unsafe"
)

type RulestoneServer struct {
	ruleEngines []*RuleEngineInfo
	conn        net.Conn
	matchActor  *actors.Actor
}

func NewRulestoneServer(conn net.Conn) *RulestoneServer {
	result := RulestoneServer{
		ruleEngines: make([]*RuleEngineInfo, 0),
		conn:        conn,
		matchActor:  actors.NewActor(nil, 10000)}
	return &result
}

type RuleEngineInfo struct {
	RuleEngineId int16
	repo         *engine.RuleEngineRepo
	api          *api.RuleApi
	ctx          *types.AppContext
	ruleEngine   *engine.RuleEngine
}

func (rs *RulestoneServer) NewRulestoneEngine() *RuleEngineInfo {
	result := RuleEngineInfo{RuleEngineId: int16(len(rs.ruleEngines)), ctx: types.NewAppContext()}
	result.api = api.NewRuleApi(result.ctx)
	result.repo = engine.NewRuleEngineRepo(result.ctx)
	rs.ruleEngines = append(rs.ruleEngines, &result)
	return &result
}

const socketPath = "/tmp/go_sidecar.sock"
const SendBuferSize = 4 * 1024 * 1024
const RecvBuferSize = 4 * 1024 * 1024
const CreateRuleEngine = 1
const AddRuleFromJsonStringCommand = 2
const AddRuleFromYamlStringCommand = 3
const AddRuleFromFileCommand = 4
const AddRulesFromDirectoryCommand = 5
const ActivateCommand = 6
const MatchCommand = 7

func main() {
	if _, err := os.Stat(socketPath); err == nil {
		os.Remove(socketPath)
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			panic(err)
		}
		rulestoneServer := NewRulestoneServer(conn)
		go rulestoneServer.handleConnection()
		fmt.Println("Connection closed", conn)
	}
}

func (rs *RulestoneServer) handleConnection() {
	defer rs.conn.Close()

	unixConn, ok := rs.conn.(*net.UnixConn)
	if !ok {
		fmt.Println("Connection is not a Unix domain socket connection")
		return
	}

	// Set the send and receive buffer sizes
	err := unixConn.SetWriteBuffer(SendBuferSize)
	if err != nil {
		fmt.Println("Error setting write buffer size:", err)
	}

	err = unixConn.SetReadBuffer(RecvBuferSize)
	if err != nil {
		fmt.Println("Error setting read buffer size:", err)
	}

	for {
		command, err := readInt16(rs.conn)
		if err != nil {
			if err == io.EOF {
				// Connection was closed by the client
				fmt.Println("Connection was closed by the client:", err)
				return
			}
			// Handle other errors, for example by logging them
			fmt.Println("Error reading message:", err)
			return
		}

		// Handle the command
		switch command {
		case CreateRuleEngine:
			ruleEngineID := rs.CreateNewRuleEngine()
			writeInt16(rs.conn, uint16(ruleEngineID))
		case ActivateCommand:
			ruleEngineID, err := readInt16(rs.conn)
			if err != nil {
				// Handle other errors, for example by logging them
				fmt.Println("Error reading message:", err)
				return
			}
			rs.ActivateRuleEngine(int(ruleEngineID))
		case AddRuleFromJsonStringCommand:
			ruleEngineID, err := readInt16(rs.conn)
			if err != nil {
				// Handle other errors, for example by logging them
				fmt.Println("Error reading message:", err)
				return
			}
			ruleString, err := readLengthPrefixedMessage(rs.conn)
			if err != nil {
				// Handle other errors, for example by logging them
				fmt.Println("Error reading message:", err)
				return
			}
			ruleId := rs.AddRuleFromString(int(ruleEngineID), ruleString, "json")
			writeInt32(rs.conn, uint32(ruleId))
		case AddRuleFromYamlStringCommand:
			ruleEngineID, err := readInt16(rs.conn)
			if err != nil {
				// Handle other errors, for example by logging them
				fmt.Println("Error reading message:", err)
				return
			}
			ruleString, err := readLengthPrefixedMessage(rs.conn)
			if err != nil {
				// Handle other errors, for example by logging them
				fmt.Println("Error reading message:", err)
				return
			}
			ruleId := rs.AddRuleFromString(int(ruleEngineID), ruleString, "json")
			writeInt32(rs.conn, uint32(ruleId))
		case AddRulesFromDirectoryCommand:
			ruleEngineID, err := readInt16(rs.conn)
			if err != nil {
				// Handle other errors, for example by logging them
				fmt.Println("Error reading message:", err)
				return
			}
			rulePath, err := readLengthPrefixedMessage(rs.conn)
			if err != nil {
				// Handle other errors, for example by logging them
				fmt.Println("Error reading message:", err)
				return
			}
			numRules := rs.AddRulesFromDirectory(int(ruleEngineID), rulePath)
			writeInt32(rs.conn, uint32(numRules))
		case AddRuleFromFileCommand:
			ruleEngineID, err := readInt16(rs.conn)
			if err != nil {
				// Handle other errors, for example by logging them
				fmt.Println("Error reading message:", err)
				return
			}
			rulePath, err := readLengthPrefixedMessage(rs.conn)
			if err != nil {
				// Handle other errors, for example by logging them
				fmt.Println("Error reading message:", err)
				return
			}
			ruleId := rs.AddRuleFromFile(int(ruleEngineID), rulePath)
			writeInt32(rs.conn, uint32(ruleId))
		case MatchCommand:
			if rs.processMatchCommand() != nil {
				return
			}
		}
	}
}

func (rs *RulestoneServer) processMatchCommand() error {
	ruleEngineID, err := readInt16(rs.conn)
	if err != nil {
		// Handle other errors, for example by logging them
		fmt.Println("Error reading message:", err)
		return err
	}
	/*
		requestId, err := readInt32(conn)
		if err != nil {
			// Handle other errors, for example by logging them
			fmt.Println("Error reading message:", err)
			return
		}

	*/
	jsonData, err := readLengthPrefixedMessage(rs.conn)
	return rs.PerformMatch(int(ruleEngineID), jsonData)
}

func readInt32(conn net.Conn) (int32, error) {
	lengthBuf := make([]byte, 4)
	_, err := io.ReadFull(conn, lengthBuf)
	if err != nil {
		return -1, err
	}
	return int32(binary.BigEndian.Uint32(lengthBuf)), nil
}

func readInt16(conn net.Conn) (int16, error) {
	v := make([]byte, 2)
	_, err := io.ReadFull(conn, v)
	if err != nil {
		return -1, err
	}
	return int16(binary.BigEndian.Uint16(v)), nil
}

// readLengthPrefixedMessage reads a length-prefixed string message from the connection.
func readLengthPrefixedMessage(conn net.Conn) (string, error) {
	length, err := readInt32(conn)
	if err != nil {
		return "", err
	}

	data := make([]byte, length)
	_, err = io.ReadFull(conn, data)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func writeInt32(conn net.Conn, v uint32) {
	buf32 := []byte{0, 0, 0, 0}
	binary.BigEndian.PutUint32(buf32, v)
	conn.Write(buf32)
}

func writeInt16(conn net.Conn, v uint16) {
	buf16 := []byte{0, 0}
	binary.BigEndian.PutUint16(buf16, v)
	conn.Write(buf16)
}

// writeLengthPrefixedMessage writes a length-prefixed string message to the connection.
func writeMatchesList(conn net.Conn, matches []condition.RuleIdType) {
	numMatches := len(matches)

	writeInt32(conn, uint32(len(matches)))

	if numMatches > 0 {
		byteSlice := (*[1 << 30]byte)(unsafe.Pointer(&matches[0]))[:len(matches)*4]
		conn.Write(byteSlice)
	}
}

func (rs *RulestoneServer) AddRulesFromDirectory(id int, rulesPath string) int {
	// Placeholder. This should initialize your rule engine and return an ID.
	fmt.Println("Initializing rule engine with rules from:", rulesPath)

	files, err := os.ReadDir(rulesPath)
	if err != nil {
		return -1
	}

	for _, file := range files {
		rulePath := filepath.Join(rulesPath, file.Name())
		rule1, err := utils.ReadRuleFromFile(rulePath, rs.ruleEngines[id].ctx)
		if err != nil {
			return -1
		}
		fd1, err := rs.ruleEngines[id].api.RuleToRuleDefinition(rule1)
		if err != nil {
			return -1
		}
		rs.ruleEngines[id].repo.Register(fd1)
	}

	return len(rs.ruleEngines[id].repo.Rules)
}

func (rs *RulestoneServer) ActivateRuleEngine(id int) int {
	fmt.Println("Activating rule engine")

	newEngine, err := engine.NewRuleEngine(rs.ruleEngines[id].repo)
	if err != nil {
		return -1
	}
	newId := len(rs.ruleEngines)
	rs.ruleEngines[id].ruleEngine = newEngine

	return newId
}

func (rs *RulestoneServer) AddRuleFromFile(id int, rulePath string) int {
	ruleString, err := utils.ReadRuleFromFile(rulePath, rs.ruleEngines[id].ctx)
	if err != nil {
		return -1
	}
	fd1, err := rs.ruleEngines[id].api.RuleToRuleDefinition(ruleString)
	if err != nil {
		return -1
	}
	return int(rs.ruleEngines[id].repo.Register(fd1))
}

func (rs *RulestoneServer) AddRuleFromString(id int, rule string, format string) int {
	ruleString, err := utils.ReadRuleFromString(rule, format, rs.ruleEngines[id].ctx)
	if err != nil {
		return -1
	}
	fd1, err := rs.ruleEngines[id].api.RuleToRuleDefinition(ruleString)
	if err != nil {
		return -1
	}
	return int(rs.ruleEngines[id].repo.Register(fd1))
}

// PerformMatch - Use the rule engine to match against the provided JSON
func (rs *RulestoneServer) PerformMatch(ruleEngineID int, jsonData string) error {
	var decoded interface{}
	err := json.Unmarshal([]byte(jsonData), &decoded)
	if err != nil {
		return err
	}

	if ruleEngineID >= len(rs.ruleEngines) {
		return errors.New("invalid rule engine ID")
	}

	ruleEngine := rs.ruleEngines[ruleEngineID].ruleEngine

	rs.matchActor.Do(func(actor *actors.Actor) {
		matches := ruleEngine.MatchEvent(decoded)
		//writeInt32(conn, uint32(requestId))
		writeMatchesList(rs.conn, matches)
	})
	return nil
}

func (rs *RulestoneServer) CreateNewRuleEngine() int16 {
	eng := rs.NewRulestoneEngine()
	return eng.RuleEngineId
}
