/*
 * Copyright 2021-2022 by Nedim Sabic Sabic
 * https://www.fibratus.io
 * All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *  http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package http

import (
	"compress/gzip"
	"encoding/json"
	"github.com/rabbitstack/fibratus/pkg/outputs"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	htypes "github.com/rabbitstack/fibratus/pkg/handle/types"
	"github.com/rabbitstack/fibratus/pkg/kevent"
	"github.com/rabbitstack/fibratus/pkg/kevent/kparams"
	"github.com/rabbitstack/fibratus/pkg/kevent/ktypes"
	pstypes "github.com/rabbitstack/fibratus/pkg/ps/types"
	shandle "github.com/rabbitstack/fibratus/pkg/syscall/handle"
	"github.com/stretchr/testify/require"
)

func TestHttpPublish(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:8081")
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/intake", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var kevents []*kevent.Kevent
		if err := json.Unmarshal(body, &kevents); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		assert.Equal(t, 3, len(kevents))
		assert.Equal(t, "aaabbbaaa", r.Header.Get("API-Key"))
		assert.Equal(t, "fibratus/", r.Header.Get("User-Agent"))
		assert.Equal(t, "1.1", r.Header.Get("Version"))
		assert.Equal(t, "Basic dXNlcjpwYXNz", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	})

	srv := httptest.NewUnstartedServer(mux)

	srv.Listener.Close()
	srv.Listener = l

	srv.Start()
	defer srv.Close()

	c := Config{
		Timeout: time.Second * 3,
		Headers: map[string]string{
			"API-Key": "aaabbbaaa",
			"Version": "1.1",
		},
		Username:   "user",
		Password:   "pass",
		Serializer: outputs.JSON,
	}

	httpClient, err := newHTTPClient(c)
	require.NoError(t, err)

	h := _http{config: c, client: httpClient, url: "http://127.0.0.1:8081/intake"}

	err = h.Publish(getBatch())
	require.NoError(t, err)
}

func TestHttpGzipPublish(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:8081")
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/intake", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "gzip", r.Header.Get("Content-Encoding"))
		gr, err := gzip.NewReader(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		body, err := io.ReadAll(gr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var kevents []*kevent.Kevent
		if err := json.Unmarshal(body, &kevents); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		assert.Equal(t, 3, len(kevents))
		w.WriteHeader(http.StatusOK)
	})

	srv := httptest.NewUnstartedServer(mux)

	srv.Listener.Close()
	srv.Listener = l

	srv.Start()
	defer srv.Close()

	c := Config{
		Timeout:    time.Second * 3,
		EnableGzip: true,
		Serializer: outputs.JSON,
	}

	httpClient, err := newHTTPClient(c)
	require.NoError(t, err)

	h := _http{config: c, client: httpClient, url: "http://127.0.0.1:8081/intake"}

	err = h.Publish(getBatch())
	require.NoError(t, err)
}

func getBatch() *kevent.Batch {
	kevt := &kevent.Kevent{
		Type:        ktypes.CreateFile,
		Tid:         2484,
		PID:         859,
		CPU:         1,
		Seq:         2,
		Name:        "CreateFile",
		Timestamp:   time.Now(),
		Category:    ktypes.File,
		Host:        "archrabbit",
		Description: "Creates or opens a new file, directory, I/O device, pipe, console",
		Kparams: kevent.Kparams{
			kparams.FileObject:    {Name: kparams.FileObject, Type: kparams.Uint64, Value: uint64(12456738026482168384)},
			kparams.FileName:      {Name: kparams.FileName, Type: kparams.UnicodeString, Value: "\\Device\\HarddiskVolume2\\Windows\\system32\\user32.dll"},
			kparams.FileType:      {Name: kparams.FileType, Type: kparams.AnsiString, Value: "file"},
			kparams.FileOperation: {Name: kparams.FileOperation, Type: kparams.AnsiString, Value: "open"},
			kparams.BasePrio:      {Name: kparams.BasePrio, Type: kparams.Int8, Value: int8(2)},
			kparams.PagePrio:      {Name: kparams.PagePrio, Type: kparams.Uint8, Value: uint8(2)},
		},
		Metadata: map[kevent.MetadataKey]any{"foo": "bar", "fooz": "baarz"},
		PS: &pstypes.PS{
			PID:       2436,
			Ppid:      6304,
			Name:      "firefox.exe",
			Exe:       `C:\Program Files\Mozilla Firefox\firefox.exe`,
			Comm:      `C:\Program Files\Mozilla Firefox\firefox.exe -contentproc --channel="6304.3.1055809391\1014207667" -childID 1 -isForBrowser -prefsHandle 2584 -prefMapHandle 2580 -prefsLen 70 -prefMapSize 216993 -parentBuildID 20200107212822 -greomni "C:\Program Files\Mozilla Firefox\omni.ja" -appomni "C:\Program Files\Mozilla Firefox\browser\omni.ja" -appdir "C:\Program Files\Mozilla Firefox\browser" - 6304 "\\.\pipe\gecko-crash-server-pipe.6304" 2596 tab`,
			Cwd:       `C:\Program Files\Mozilla Firefox\`,
			SID:       "archrabbit\\SYSTEM",
			Args:      []string{"-contentproc", `--channel=6304.3.1055809391\1014207667`, "-childID", "1", "-isForBrowser", "-prefsHandle", "2584", "-prefMapHandle", "2580", "-prefsLen", "70", "-prefMapSize", "216993", "-parentBuildID"},
			SessionID: 4,
			Envs:      map[string]string{"ProgramData": "C:\\ProgramData", "COMPUTRENAME": "archrabbit"},
			Threads: map[uint32]pstypes.Thread{
				3453: {Tid: 3453, Entrypoint: kparams.Hex("0x7ffe2557ff80"), IOPrio: 2, PagePrio: 5, KstackBase: kparams.Hex("0xffffc307810d6000"), KstackLimit: kparams.Hex("0xffffc307810cf000"), UstackLimit: kparams.Hex("0x5260000"), UstackBase: kparams.Hex("0x525f000")},
				3455: {Tid: 3455, Entrypoint: kparams.Hex("0x5efe2557ff80"), IOPrio: 3, PagePrio: 5, KstackBase: kparams.Hex("0xffffc307810d6000"), KstackLimit: kparams.Hex("0xffffc307810cf000"), UstackLimit: kparams.Hex("0x5260000"), UstackBase: kparams.Hex("0x525f000")},
			},
			Handles: []htypes.Handle{
				{Num: shandle.Handle(0xffffd105e9baaf70),
					Name:   `\REGISTRY\MACHINE\SYSTEM\ControlSet001\Services\Tcpip\Parameters\Interfaces\{b677c565-6ca5-45d3-b618-736b4e09b036}`,
					Type:   "Key",
					Object: 777488883434455544,
					Pid:    uint32(1023),
				},
				{
					Num:  shandle.Handle(0xffffd105e9adaf70),
					Name: `\RPC Control\OLEA61B27E13E028C4EA6C286932E80`,
					Type: "ALPC Port",
					Pid:  uint32(1023),
					MD: &htypes.AlpcPortInfo{
						Seqno:   1,
						Context: 0x0,
						Flags:   0x0,
					},
					Object: 457488883434455544,
				},
				{
					Num:  shandle.Handle(0xeaffd105e9adaf30),
					Name: `C:\Users\bunny`,
					Type: "File",
					Pid:  uint32(1023),
					MD: &htypes.FileInfo{
						IsDirectory: true,
					},
					Object: 357488883434455544,
				},
			},
		},
	}

	kevt1 := &kevent.Kevent{
		Type:        ktypes.CreateFile,
		Tid:         2484,
		PID:         459,
		CPU:         1,
		Seq:         2,
		Name:        "CreateFile",
		Timestamp:   time.Now(),
		Category:    ktypes.File,
		Host:        "archrabbit",
		Description: "Creates or opens a new file, directory, I/O device, pipe, console",
		Kparams: kevent.Kparams{
			kparams.FileObject:    {Name: kparams.FileObject, Type: kparams.Uint64, Value: uint64(12456738026482168384)},
			kparams.FileName:      {Name: kparams.FileName, Type: kparams.UnicodeString, Value: "\\Device\\HarddiskVolume2\\Windows\\system32\\user32.dll"},
			kparams.FileType:      {Name: kparams.FileType, Type: kparams.AnsiString, Value: "file"},
			kparams.FileOperation: {Name: kparams.FileOperation, Type: kparams.AnsiString, Value: "open"},
			kparams.BasePrio:      {Name: kparams.BasePrio, Type: kparams.Int8, Value: int8(2)},
			kparams.PagePrio:      {Name: kparams.PagePrio, Type: kparams.Uint8, Value: uint8(2)},
		},
		Metadata: map[kevent.MetadataKey]any{"foo": "bar", "fooz": "baarz"},
		PS: &pstypes.PS{
			PID:       2436,
			Ppid:      6304,
			Name:      "firefox.exe",
			Exe:       `C:\Program Files\Mozilla Firefox\firefox.exe`,
			Comm:      `C:\Program Files\Mozilla Firefox\firefox.exe -contentproc --channel="6304.3.1055809391\1014207667" -childID 1 -isForBrowser -prefsHandle 2584 -prefMapHandle 2580 -prefsLen 70 -prefMapSize 216993 -parentBuildID 20200107212822 -greomni "C:\Program Files\Mozilla Firefox\omni.ja" -appomni "C:\Program Files\Mozilla Firefox\browser\omni.ja" -appdir "C:\Program Files\Mozilla Firefox\browser" - 6304 "\\.\pipe\gecko-crash-server-pipe.6304" 2596 tab`,
			Cwd:       `C:\Program Files\Mozilla Firefox\`,
			SID:       "archrabbit\\SYSTEM",
			Args:      []string{"-contentproc", `--channel=6304.3.1055809391\1014207667`, "-childID", "1", "-isForBrowser", "-prefsHandle", "2584", "-prefMapHandle", "2580", "-prefsLen", "70", "-prefMapSize", "216993", "-parentBuildID"},
			SessionID: 4,
			Envs:      map[string]string{"ProgramData": "C:\\ProgramData", "COMPUTRENAME": "archrabbit"},
			Threads: map[uint32]pstypes.Thread{
				3453: {Tid: 3453, Entrypoint: kparams.Hex("0x7ffe2557ff80"), IOPrio: 2, PagePrio: 5, KstackBase: kparams.Hex("0xffffc307810d6000"), KstackLimit: kparams.Hex("0xffffc307810cf000"), UstackLimit: kparams.Hex("0x5260000"), UstackBase: kparams.Hex("0x525f000")},
				3455: {Tid: 3455, Entrypoint: kparams.Hex("0x5efe2557ff80"), IOPrio: 3, PagePrio: 5, KstackBase: kparams.Hex("0xffffc307810d6000"), KstackLimit: kparams.Hex("0xffffc307810cf000"), UstackLimit: kparams.Hex("0x5260000"), UstackBase: kparams.Hex("0x525f000")},
			},
			Handles: []htypes.Handle{
				{Num: shandle.Handle(0xffffd105e9baaf70),
					Name:   `\REGISTRY\MACHINE\SYSTEM\ControlSet001\Services\Tcpip\Parameters\Interfaces\{b677c565-6ca5-45d3-b618-736b4e09b036}`,
					Type:   "Key",
					Object: 777488883434455544,
					Pid:    uint32(1023),
				},
				{
					Num:  shandle.Handle(0xffffd105e9adaf70),
					Name: `\RPC Control\OLEA61B27E13E028C4EA6C286932E80`,
					Type: "ALPC Port",
					Pid:  uint32(1023),
					MD: &htypes.AlpcPortInfo{
						Seqno:   1,
						Context: 0x0,
						Flags:   0x0,
					},
					Object: 457488883434455544,
				},
				{
					Num:  shandle.Handle(0xeaffd105e9adaf30),
					Name: `C:\Users\bunny`,
					Type: "File",
					Pid:  uint32(1023),
					MD: &htypes.FileInfo{
						IsDirectory: true,
					},
					Object: 357488883434455544,
				},
			},
		},
	}

	kevt2 := &kevent.Kevent{
		Type:        ktypes.CreateFile,
		Tid:         2484,
		PID:         829,
		CPU:         1,
		Seq:         2,
		Name:        "CreateFile",
		Timestamp:   time.Now(),
		Category:    ktypes.File,
		Host:        "archrabbit",
		Description: "Creates or opens a new file, directory, I/O device, pipe, console",
		Kparams: kevent.Kparams{
			kparams.FileObject:    {Name: kparams.FileObject, Type: kparams.Uint64, Value: uint64(12456738026482168384)},
			kparams.FileName:      {Name: kparams.FileName, Type: kparams.UnicodeString, Value: "\\Device\\HarddiskVolume2\\Windows\\system32\\user32.dll"},
			kparams.FileType:      {Name: kparams.FileType, Type: kparams.AnsiString, Value: "file"},
			kparams.FileOperation: {Name: kparams.FileOperation, Type: kparams.AnsiString, Value: "open"},
			kparams.BasePrio:      {Name: kparams.BasePrio, Type: kparams.Int8, Value: int8(2)},
			kparams.PagePrio:      {Name: kparams.PagePrio, Type: kparams.Uint8, Value: uint8(2)},
		},
		Metadata: map[kevent.MetadataKey]any{"foo": "bar", "fooz": "baarz"},
		PS: &pstypes.PS{
			PID:       829,
			Ppid:      6304,
			Name:      "firefox.exe",
			Exe:       `C:\Program Files\Mozilla Firefox\firefox.exe`,
			Comm:      `C:\Program Files\Mozilla Firefox\firefox.exe -contentproc --channel="6304.3.1055809391\1014207667" -childID 1 -isForBrowser -prefsHandle 2584 -prefMapHandle 2580 -prefsLen 70 -prefMapSize 216993 -parentBuildID 20200107212822 -greomni "C:\Program Files\Mozilla Firefox\omni.ja" -appomni "C:\Program Files\Mozilla Firefox\browser\omni.ja" -appdir "C:\Program Files\Mozilla Firefox\browser" - 6304 "\\.\pipe\gecko-crash-server-pipe.6304" 2596 tab`,
			Cwd:       `C:\Program Files\Mozilla Firefox\`,
			SID:       "archrabbit\\SYSTEM",
			Args:      []string{"-contentproc", `--channel=6304.3.1055809391\1014207667`, "-childID", "1", "-isForBrowser", "-prefsHandle", "2584", "-prefMapHandle", "2580", "-prefsLen", "70", "-prefMapSize", "216993", "-parentBuildID"},
			SessionID: 4,
			Envs:      map[string]string{"ProgramData": "C:\\ProgramData", "COMPUTRENAME": "archrabbit"},
			Threads: map[uint32]pstypes.Thread{
				3453: {Tid: 3453, Entrypoint: kparams.Hex("0x7ffe2557ff80"), IOPrio: 2, PagePrio: 5, KstackBase: kparams.Hex("0xffffc307810d6000"), KstackLimit: kparams.Hex("0xffffc307810cf000"), UstackLimit: kparams.Hex("0x5260000"), UstackBase: kparams.Hex("0x525f000")},
				3455: {Tid: 3455, Entrypoint: kparams.Hex("0x5efe2557ff80"), IOPrio: 3, PagePrio: 5, KstackBase: kparams.Hex("0xffffc307810d6000"), KstackLimit: kparams.Hex("0xffffc307810cf000"), UstackLimit: kparams.Hex("0x5260000"), UstackBase: kparams.Hex("0x525f000")},
			},
			Handles: []htypes.Handle{
				{Num: shandle.Handle(0xffffd105e9baaf70),
					Name:   `\REGISTRY\MACHINE\SYSTEM\ControlSet001\Services\Tcpip\Parameters\Interfaces\{b677c565-6ca5-45d3-b618-736b4e09b036}`,
					Type:   "Key",
					Object: 777488883434455544,
					Pid:    uint32(1023),
				},
				{
					Num:  shandle.Handle(0xffffd105e9adaf70),
					Name: `\RPC Control\OLEA61B27E13E028C4EA6C286932E80`,
					Type: "ALPC Port",
					Pid:  uint32(1023),
					MD: &htypes.AlpcPortInfo{
						Seqno:   1,
						Context: 0x0,
						Flags:   0x0,
					},
					Object: 457488883434455544,
				},
				{
					Num:  shandle.Handle(0xeaffd105e9adaf30),
					Name: `C:\Users\bunny`,
					Type: "File",
					Pid:  uint32(1023),
					MD: &htypes.FileInfo{
						IsDirectory: true,
					},
					Object: 357488883434455544,
				},
			},
		},
	}

	return kevent.NewBatch(kevt, kevt1, kevt2)
}
