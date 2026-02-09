package plugin

import (
	"encoding/binary"
	"encoding/json"
	"io"
	"slices"
	"sync"
)

type ServiceCodeWithSourceMap struct {
	ServiceText []byte
	ScriptKind ScriptKind
	Mappings []byte
}

type PluginOptions struct {
	Input io.Reader
	Output io.Writer
	// Example: [".vue"]
	ExtraExtensions []string
	CreateServiceCodeWithSourceMap func (fileName string, sourceText string) *ServiceCodeWithSourceMap
}

func ensureCap(b []byte, needed uint32) []byte {
	if b == nil || uint32(cap(b)) < needed {
		b = make([]byte, needed)
	}
	return b[:needed]
}

type ScriptKind uint8

const (
	ScriptKindJS ScriptKind = iota
	ScriptKindJSX
	ScriptKindTS
	ScriptKindTSX
)

type MsgKind uint8

const (
	MsgKindCreateServiceCode MsgKind = iota
	MsgKindCreateServiceCodeResponse
)

type InitializationMessage struct {
	ExtraExtensions []string `json:"extraExtensions"`
}

func Run(opts PluginOptions) {
	var header [5]byte
	var recvBuf []byte

	{
		if opts.ExtraExtensions == nil {
			opts.ExtraExtensions = []string{}
		}
		initialization, err := json.Marshal(InitializationMessage{
			ExtraExtensions: opts.ExtraExtensions,
		})
		if err != nil {
			panic(err)
		}

		binary.LittleEndian.PutUint32(header[:], uint32(len(initialization)))
		if _, err = opts.Output.Write(header[:4]); err != nil {
			panic(err)
		}
		if _, err = opts.Output.Write(initialization); err != nil {
			panic(err)
		}
	}

	tasks := make(chan []byte, 1000)
	var writeMu sync.Mutex
	for range 4 {
		go func() {
			var sendBuf []byte
			for task := range tasks {
				offset := uint32(0)
				reqId := binary.LittleEndian.Uint64(task[offset:])
				offset += 8
				fileNameLen := binary.LittleEndian.Uint32(task[offset:])
				offset += 4
				fileName := string(task[offset:offset+fileNameLen])
				offset += fileNameLen
				sourceTextLen := binary.LittleEndian.Uint32(task[offset:])
				offset += 4
				sourceText := string(task[offset:offset+sourceTextLen])
				offset += sourceTextLen

				res := opts.CreateServiceCodeWithSourceMap(fileName, sourceText)

				responsePayloadLen := uint32(8 + 1 + 1 + 4 + len(res.ServiceText) + 4 + len(res.Mappings))

				offset = 0
				sendBuf = ensureCap(sendBuf, 5 + responsePayloadLen)
				sendBuf[0] = byte(MsgKindCreateServiceCodeResponse)
				offset += 1
				binary.LittleEndian.PutUint32(sendBuf[offset:], responsePayloadLen)
				offset += 4
				binary.LittleEndian.PutUint64(sendBuf[offset:], reqId)
				offset += 8
				sendBuf[offset] = 1 // TODO
				offset += 1
				sendBuf[offset] = byte(res.ScriptKind)
				offset += 1
				binary.LittleEndian.PutUint32(sendBuf[offset:], uint32(len(res.ServiceText)))
				offset += 4
				copy(sendBuf[offset:], res.ServiceText)
				offset += uint32(len(res.ServiceText))
				binary.LittleEndian.PutUint32(sendBuf[offset:], uint32(len(res.Mappings)))
				offset += 4
				copy(sendBuf[offset:], res.Mappings)
				offset += uint32(len(res.Mappings))

				writeMu.Lock()
				_, err := opts.Output.Write(sendBuf)
				writeMu.Unlock()
				if err != nil {
					panic(err)
				}
			}
		}()
	}
	for {
		_, err := io.ReadFull(opts.Input, header[:])
		if err == io.EOF {
			return
		} else if err != nil {
			panic(err)
		}

		msgKind := MsgKind(header[0])
		payloadLen := binary.LittleEndian.Uint32(header[1:])
		recvBuf = ensureCap(recvBuf, payloadLen)

		if _, err := io.ReadFull(opts.Input, recvBuf); err != nil {
			panic(err)
		}

		switch msgKind {
		case MsgKindCreateServiceCode:
			tasks <- slices.Clone(recvBuf[:payloadLen])
		}
	}
}
