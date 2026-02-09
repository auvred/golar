package pluginhost

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/auvred/golar/internal/mapping"
	"github.com/auvred/golar/plugin"
	"github.com/microsoft/typescript-go/shim/core"
)

type Plugin struct {
	stdin      io.WriteCloser
	stdout     io.ReadCloser
	sendBuf []byte
	mu sync.Mutex

	reqId atomic.Uint64
	createServiceCodeRequests sync.Map

	ExtraExtensions []string
}

func NewPlugin(args []string) (*Plugin, error) {
	p := Plugin{}
	cmd := exec.Command(args[0], args[1:]...)
	var err error
	p.stdin, err = cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stdin pipe: %v", err)
	}
	p.stdout, err = cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stdout pipe: %v", err)
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	var header [5]byte
	var recvBuf []byte
	if _, err := io.ReadFull(p.stdout, header[:4]); err != nil {
		panic(err)
	}
	payloadLen := binary.LittleEndian.Uint32(header[:])
	recvBuf = ensureCap(recvBuf, payloadLen)
	if _, err := io.ReadFull(p.stdout, recvBuf); err != nil {
		panic(err)
	}
	initialization := plugin.InitializationMessage{}
	if err := json.Unmarshal(recvBuf, &initialization); err != nil {
		panic(err)
	}
	p.ExtraExtensions = initialization.ExtraExtensions

	go func() {
		for {
			_, err := io.ReadFull(p.stdout, header[:])
			if err != nil {
				// TODO?
				// if err == io.EOF {
				// 	return
				// }
				panic(err)
			}
			msgKind := plugin.MsgKind(header[0])
			payloadLen := binary.LittleEndian.Uint32(header[1:])
			recvBuf = ensureCap(recvBuf, payloadLen)
			if _, err := io.ReadFull(p.stdout, recvBuf); err != nil {
				panic(err)
			}
			switch msgKind {
			case plugin.MsgKindCreateServiceCodeResponse:
				reqId := binary.LittleEndian.Uint64(recvBuf)
				f, _ := p.createServiceCodeRequests.LoadAndDelete(reqId)
				f.(func (r[]byte))(recvBuf[8:])
			}
		}
	}()

	return &p, nil
}

func (p *Plugin) sendMessage(msgKind plugin.MsgKind, payload []byte) error {
	var header [5]byte
	header[0] = byte(msgKind)
	binary.LittleEndian.PutUint32(header[1:], uint32(len(payload)))
	_, err := p.stdin.Write(header[:])
	if err != nil {
		return err
	}
	_, err = p.stdin.Write(payload)
	return err
}

func ensureCap(b []byte, needed uint32) []byte {
	if b == nil || uint32(cap(b)) < needed {
		b = make([]byte, needed)
	}
	return b[:needed]
}

type CreateServiceCodeResponse struct {
	ServiceText string
	SourceMap string
	ScriptKind core.ScriptKind
	Mappings []mapping.Mapping
	IgnoreMappings []mapping.IgnoreDirectiveMapping
}

func (p *Plugin) CreateServiceCode(fileName string, sourceText string) <- chan CreateServiceCodeResponse {
	ch := make(chan CreateServiceCodeResponse, 1)

	reqId := p.reqId.Add(1)
	p.createServiceCodeRequests.Store(reqId, func (payload []byte) {
		offset := uint32(0)

		properties := payload[offset]
		offset += 1
		scriptKind := plugin.ScriptKind(payload[offset])
		offset += 1

		response := CreateServiceCodeResponse{}
		switch scriptKind {
		case plugin.ScriptKindJS:
			response.ScriptKind = core.ScriptKindJS
		case plugin.ScriptKindJSX:
			response.ScriptKind = core.ScriptKindJSX
		case plugin.ScriptKindTS:
			response.ScriptKind = core.ScriptKindTS
		case plugin.ScriptKindTSX:
			response.ScriptKind = core.ScriptKindTSX
		}

		if properties & 1 != 0 {
			serviceTextLen := binary.LittleEndian.Uint32(payload[offset:])
			offset += 4
			response.ServiceText = string(payload[offset:offset+serviceTextLen])
			offset += serviceTextLen

			sourceMapLen := binary.LittleEndian.Uint32(payload[offset:])
			offset += 4
			response.SourceMap = string(payload[offset:offset+sourceMapLen])
			offset += sourceMapLen
		} else {
			serviceTextLen := binary.LittleEndian.Uint32(payload[offset:])
			offset += 4
			response.ServiceText = string(payload[offset:offset+serviceTextLen])
			offset += serviceTextLen

			mappingsCount := binary.LittleEndian.Uint32(payload[offset:])
			mappingsByteLen := mappingsCount * uint32(unsafe.Sizeof(mapping.Mapping{}))
			offset += 4
			response.Mappings = make([]mapping.Mapping, mappingsCount)
			copy(response.Mappings, unsafe.Slice((*mapping.Mapping)(unsafe.Pointer(unsafe.SliceData(payload[offset:offset+mappingsByteLen]))), mappingsCount))
			offset += mappingsByteLen

			ignoreMappingsCount := binary.LittleEndian.Uint32(payload[offset:])
			ignoreMappingsByteLen := ignoreMappingsCount * uint32(unsafe.Sizeof(mapping.IgnoreDirectiveMapping{}))
			offset += 4
			response.IgnoreMappings = make([]mapping.IgnoreDirectiveMapping, ignoreMappingsCount)
			copy(response.IgnoreMappings, unsafe.Slice((*mapping.IgnoreDirectiveMapping)(unsafe.Pointer(unsafe.SliceData(payload[offset:offset+ignoreMappingsByteLen]))), ignoreMappingsCount))
		}

		ch <- response
	})
	p.mu.Lock()
	defer p.mu.Unlock()

	p.sendBuf = ensureCap(p.sendBuf, uint32(8 + 4 + len(fileName) + 4 + len(sourceText)))
	binary.LittleEndian.PutUint64(p.sendBuf, reqId)
	offset := 8
	binary.LittleEndian.PutUint32(p.sendBuf[offset:], uint32(len(fileName)))
	offset += 4
	copy(p.sendBuf[offset:], fileName)
	offset += len(fileName)
	binary.LittleEndian.PutUint32(p.sendBuf[offset:], uint32(len(sourceText)))
	offset += 4
	copy(p.sendBuf[offset:], sourceText)
	offset += len(sourceText)
	p.sendMessage(plugin.MsgKindCreateServiceCode, p.sendBuf)

	return ch
}
