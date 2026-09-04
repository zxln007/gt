// Copyright (c) 2022 Institute of Software, Chinese Academy of Sciences (ISCAS)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package test

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/isrc-cas/gt/client/std"
	"github.com/isrc-cas/gt/client/webrtc"
)

func TestP2PGetOffer(t *testing.T) {
	t.Parallel()

	// 创建 HTTP echo 服务
	httpListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	httpEchoServerAddr := httpListener.Addr().String()
	go httpEchoServer(httpListener)

	// 创建客户端、服务端
	s, err := setupServer([]string{
		"server",
		"-addr", "127.0.0.1:0",
		"-stunAddr", "127.0.0.1:0",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	c, err := setupClient([]string{
		"client",
		"-id", "abc",
		"-secret", "eec1eabf-2c59-4e19-bf10-34707c17ed89",
		"-local", fmt.Sprintf("http://%s", httpEchoServerAddr),
		"-remote", s.GetListenerAddrPort().String(),
		"-remoteSTUN", "stun:" + s.GetSTUNListenerAddrPort().String(),
		"-webrtcThreadMode",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	// 仅建立连接
	var peerConnection *webrtc.PeerConnection
	candidateDoneChan := make(chan struct{})
	peerConnectionConfig := webrtc.PeerConnectionConfig{
		ICEServers: []string{},
		OnSignalingChange: func(state webrtc.SignalingState) {
		},
		OnDataChannel: func(dataChannelWithoutCallback *webrtc.DataChannelWithoutCallback) {
		},
		OnRenegotiationNeeded: func() {
		},
		OnNegotiationNeeded: func() {
		},
		OnICEConnectionChange: func(state webrtc.ICEConnectionState) {
		},
		OnStandardizedICEConnectionChange: func(state webrtc.ICEConnectionState) {
		},
		OnConnectionChange: func(state webrtc.PeerConnectionState) {
		},
		OnICEGatheringChange: func(state webrtc.ICEGatheringState) {
			if state == webrtc.ICEGatheringStateComplete {
				close(candidateDoneChan)
			}
		},
		OnICECandidate: func(iceCandidate *webrtc.ICECandidate) {
		},
		OnICECandidateError: func(addrss string, port int, url string, errorCode int, errorText string) {
		},
	}
	err = webrtc.NewPeerConnection(&peerConnectionConfig, &peerConnection, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	var dataChannelUnused *webrtc.DataChannel
	err = peerConnection.CreateDataChannelWithID("only", 100, true, false, nil, &dataChannelUnused)
	if err != nil {
		t.Fatal(err)
	}

	// 获取 offer
	httpClient := setupHTTPClient(s.GetListenerAddrPort().String(), nil)
	req, err := http.NewRequest("XP", "http://abc.p2p.com/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("WebRTC-OP", "get-offer")
	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	offerLen := make([]byte, 2)
	_, err = io.ReadFull(resp.Body, offerLen)
	if err != nil {
		t.Fatal(err)
	}
	offerBytes := make([]byte, uint16(offerLen[0])<<8|uint16(offerLen[1]))
	_, err = io.ReadFull(resp.Body, offerBytes)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("get offer: %s", string(offerBytes))
	var offer webrtc.SessionDescription
	err = json.Unmarshal(offerBytes, &offer)
	if err != nil {
		t.Fatal(err)
	}
	err = peerConnection.SetRemoteDescription(&offer)
	if err != nil {
		t.Fatal(err)
	}

	// 获取 candidate 与 peer task id（响应尾部 {"id":N}）
	var id struct {
		ID uint32
	}
	for {
		var msgLen uint16
		err = binary.Read(resp.Body, binary.BigEndian, &msgLen)
		if err != nil {
			t.Fatal(err)
		}
		msg := make([]byte, msgLen)
		_, err = io.ReadFull(resp.Body, msg)
		if err != nil {
			t.Fatal(err)
		}
		var probe map[string]json.RawMessage
		if json.Unmarshal(msg, &probe) == nil {
			if _, ok := probe["id"]; ok {
				err = json.Unmarshal(msg, &id)
				if err != nil {
					t.Fatal(err)
				}
				break
			}
		}
		t.Logf("get candidate: %s", string(msg))
		var candidate webrtc.ICECandidate
		err = json.Unmarshal(msg, &candidate)
		if err != nil {
			t.Fatal(err)
		}
		err = peerConnection.AddICECandidate(&candidate)
		if err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("get id: %d", id.ID)
	err = resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}

	// 发送 answer
	answer, err := peerConnection.CreateAnswer()
	if err != nil {
		t.Fatal(err)
	}
	err = peerConnection.SetLocalDescription(answer)
	if err != nil {
		t.Fatal(err)
	}
	httpClient = setupHTTPClient(s.GetListenerAddrPort().String(), nil)
	req, err = http.NewRequest("XP", "http://abc.p2p.com/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("WebRTC-OP", "resp-answer")
	req.Header.Set("WebRTC-OP-ID", strconv.FormatUint(uint64(id.ID), 10))
	select {
	case <-candidateDoneChan:
	case <-time.After(60 * time.Second):
		t.Fatal("ice gathering timeout")
	}
	answerBuf := &bytes.Buffer{}
	chunkedWriter := std.NewChunkedWriter(answerBuf)
	answerBytes, err := json.Marshal(peerConnection.GetLocalDescription())
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("send answer: %s", string(answerBytes))
	_, err = chunkedWriter.Write([]byte{byte(len(answerBytes) >> 8), byte(len(answerBytes))})
	if err != nil {
		t.Fatal(err)
	}
	_, err = chunkedWriter.Write(answerBytes)
	if err != nil {
		t.Fatal(err)
	}
	req.Body = io.NopCloser(answerBuf)
	resp, err = httpClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	err = resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
}

func httpEchoServer(l net.Listener) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		buf, err := io.ReadAll(r.Body)
		if err != nil {
			panic(err)
		}
		_, err = w.Write(buf)
		if err != nil {
			panic(err)
		}
	})
	err := http.Serve(l, mux)
	if err != nil {
		panic(err)
	}
}

func TestP2PSetOffer(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		_, err := writer.Write([]byte("ok"))
		if err != nil {
			panic(err)
		}
	})
	httpServer := &http.Server{Handler: mux}
	httpListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := httpServer.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()
	go func() {
		err := httpServer.Serve(httpListener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()
	s, err := setupServer([]string{
		"server",
		"-addr", "127.0.0.1:0",
		"-stunAddr", "127.0.0.1:0",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	c, err := setupClient([]string{
		"client",
		"-id", "abc",
		"-secret", "eec1eabf-2c59-4e19-bf10-34707c17ed89",
		"-local", fmt.Sprintf("http://%s", httpListener.Addr().String()),
		"-remote", s.GetListenerAddrPort().String(),
		"-remoteSTUN", "stun:" + s.GetSTUNListenerAddrPort().String(),
		"-logLevel", "debug",
		"-webrtcThreadMode",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	httpClient := setupHTTPClient(s.GetListenerAddrPort().String(), nil)
	// 全量并行测试环境下信令请求可能被拖住,给出硬超时把挂死转成快速失败
	httpClient.Timeout = 20 * time.Second

	// 访客 PC 与产品网关一致使用线程池线程(生产路径 client.go 无条件创建;
	// 裸 nil 线程的兜底通路不具备 socket 分发能力,ICE 无法连通)
	threadPool := webrtc.NewThreadPool(3)
	defer threadPool.Close()
	pc, offer := initOffer(t, s.GetSTUNListenerAddrPort().String(), threadPool)

	req, err := http.NewRequest("XP", "http://abc.p2p.com/test", nil)
	if err != nil {
		t.Fatal(err)
	}
	raw := &bytes.Buffer{}
	b := std.NewChunkedWriter(raw)
	req.Body = io.NopCloser(raw)
	req.ContentLength = -1
	sdp, err := json.Marshal(offer)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("XP sdp: %s", sdp)
	sdpLen := uint16(len(sdp))
	_, err = b.Write(append([]byte{byte(sdpLen >> 8), byte(sdpLen)}, sdp...))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatal("invalid status code")
	}

	var answer webrtc.SessionDescription
	var dataLen uint16
	err = binary.Read(resp.Body, binary.BigEndian, &dataLen)
	if err != nil {
		t.Fatal(err)
	}
	answerBytes := make([]byte, dataLen)
	_, err = io.ReadFull(resp.Body, answerBytes)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("XP sdp: %s", answerBytes)
	err = json.Unmarshal(answerBytes, &answer)
	if err != nil {
		t.Fatal(err)
	}

	err = pc.SetRemoteDescription(&answer)
	if err != nil {
		t.Fatal(err)
	}

	// 获取 candidate 与 peer task id（响应尾部 {"id":N}）,后续重协商请求靠 id 路由
	var peerID struct {
		ID uint32
	}
	for {
		err = binary.Read(resp.Body, binary.BigEndian, &dataLen)
		if err != nil {
			t.Fatal(err)
		}
		msgBytes := make([]byte, dataLen)
		_, err = io.ReadFull(resp.Body, msgBytes)
		if err != nil {
			t.Fatal(err)
		}
		var probe map[string]json.RawMessage
		if json.Unmarshal(msgBytes, &probe) == nil {
			if _, ok := probe["id"]; ok {
				err = json.Unmarshal(msgBytes, &peerID)
				if err != nil {
					t.Fatal(err)
				}
				break
			}
		}
		t.Logf("XP candidate: %s", msgBytes)
		var candidate webrtc.ICECandidate
		err = json.Unmarshal(msgBytes, &candidate)
		if err != nil {
			t.Fatal(err)
		}
		err = pc.AddICECandidate(&candidate)
		if err != nil {
			t.Fatal(err)
		}
	}

	// 重协商:连接建立后创建的 in-band 通道必须经重新协商进入 SDP 才会 open。
	// 闭包返回 error 而不是 t.Fatal:t.Fatal 会 Goexit 并执行闭包里 defer 的
	// Body.Close(),该 Close 走隧道写路径,在僵死的信令连接上会无限阻塞,
	// 把测试失败变成整包挂死(15m 超时)
	renegotiate := func() error {
		renegOffer, err := pc.CreateOffer()
		if err != nil {
			return err
		}
		err = pc.SetLocalDescription(renegOffer)
		if err != nil {
			return err
		}
		renegSDP, err := json.Marshal(pc.GetLocalDescription())
		if err != nil {
			return err
		}
		renegRaw := &bytes.Buffer{}
		w := std.NewChunkedWriter(renegRaw)
		_, err = w.Write(append([]byte{byte(len(renegSDP) >> 8), byte(len(renegSDP))}, renegSDP...))
		if err != nil {
			return err
		}
		renegReq, err := http.NewRequest("XP", "http://abc.p2p.com/test", nil)
		if err != nil {
			return err
		}
		renegReq.Header.Set("WebRTC-OP-ID", strconv.FormatUint(uint64(peerID.ID), 10))
		renegReq.Body = io.NopCloser(renegRaw)
		renegReq.ContentLength = -1
		renegResp, err := httpClient.Do(renegReq)
		if err != nil {
			return err
		}
		defer renegResp.Body.Close()
		if renegResp.StatusCode != http.StatusOK {
			return errors.New("invalid renegotiation status code")
		}
		var answerLen uint16
		err = binary.Read(renegResp.Body, binary.BigEndian, &answerLen)
		if err != nil {
			return err
		}
		answerBytes := make([]byte, answerLen)
		_, err = io.ReadFull(renegResp.Body, answerBytes)
		if err != nil {
			return err
		}
		var renegAnswer webrtc.SessionDescription
		err = json.Unmarshal(answerBytes, &renegAnswer)
		if err != nil {
			return err
		}
		return pc.SetRemoteDescription(&renegAnswer)
	}

	for i := 0; i < 10; i++ {
		var dataChannel *webrtc.DataChannel
		stateChange := make(chan webrtc.DataState, 8)
		messages := make(chan []byte, 64)
		config := webrtc.DataChannelConfig{
			OnStateChange: func(state webrtc.DataState) {
				select {
				case stateChange <- state:
				default:
				}
				if state == webrtc.DataStateOpen {
					if !dataChannel.Send([]byte("GET / HTTP/1.1\r\nHost: abc.p2p.com\r\n\r\n")) {
						panic("failed to send message with data channel")
					}
				}
			},
			OnMessage: func(message []byte) {
				select {
				case messages <- message:
				default:
				}
			},
		}
		err = pc.CreateDataChannelWithID(fmt.Sprintf("test%d", i), 0, false, true, &config, &dataChannel)
		if err != nil {
			t.Fatal(err)
		}
		if err := renegotiate(); err != nil {
			t.Fatalf("renegotiate channel %d: %v", i, err)
		}
		deadline := time.After(30 * time.Second)
	openWait:
		for {
			select {
			case state := <-stateChange:
				if state == webrtc.DataStateOpen {
					break openWait
				}
			case <-deadline:
				t.Fatal("data channel is not open after renegotiation")
			}
		}
		var respBuf []byte
	bodyWait:
		for {
			select {
			case msg := <-messages:
				respBuf = append(respBuf, msg...)
				if bytes.Contains(respBuf, []byte("ok")) {
					break bodyWait
				}
			case <-deadline:
				t.Fatalf("no response over data channel: %s", string(respBuf))
			}
		}
		t.Logf("channel %d received: %s", i, string(respBuf))
		dataChannel.Close()
	}
	t.Log("XP done")
	s.Shutdown()
}

func initOffer(t *testing.T, addr string, threadPool *webrtc.ThreadPool) (*webrtc.PeerConnection, *webrtc.SessionDescription) {
	waitNegotiationNeeded := make(chan struct{})
	var peerConnection *webrtc.PeerConnection
	candidateDoneChan := make(chan struct{})
	config := webrtc.PeerConnectionConfig{
		ICEServers: []string{
			fmt.Sprintf("stun:%s", addr),
		},
		OnSignalingChange: func(state webrtc.SignalingState) {
		},
		OnDataChannel: func(dataChannelWithoutCallback *webrtc.DataChannelWithoutCallback) {
		},
		OnRenegotiationNeeded: func() {
		},
		OnNegotiationNeeded: func() {
			close(waitNegotiationNeeded)
		},
		OnICEConnectionChange: func(state webrtc.ICEConnectionState) {
			fmt.Println("ice connection", state.String())
		},
		OnStandardizedICEConnectionChange: func(state webrtc.ICEConnectionState) {
		},
		OnConnectionChange: func(state webrtc.PeerConnectionState) {
		},
		OnICEGatheringChange: func(state webrtc.ICEGatheringState) {
			if state == webrtc.ICEGatheringStateComplete {
				close(candidateDoneChan)
			}
		},
		OnICECandidate: func(iceCandidate *webrtc.ICECandidate) {
		},
		OnICECandidateError: func(addrss string, port int, url string, errorCode int, errorText string) {
		},
	}
	var err error
	err = webrtc.NewPeerConnection(&config, &peerConnection, threadPool.GetThread(), threadPool.GetSocketThread(), threadPool.GetThread())
	if err != nil {
		t.Fatal(err)
	}

	var dataChannelUnused *webrtc.DataChannel
	err = peerConnection.CreateDataChannelWithID("only", 100, true, false, nil, &dataChannelUnused)
	if err != nil {
		t.Fatal(err)
	}

	// 带超时等待:全量并行下 libwebrtc 回调可能被拖慢,不能无限期阻塞
	select {
	case <-waitNegotiationNeeded:
	case <-time.After(60 * time.Second):
		t.Fatal("negotiation needed timeout")
	}
	offer, err := peerConnection.CreateOffer()
	if err != nil {
		t.Fatal(err)
	}
	err = peerConnection.SetLocalDescription(offer)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-candidateDoneChan:
	case <-time.After(60 * time.Second):
		t.Fatal("ice gathering timeout")
	}

	return peerConnection, peerConnection.GetLocalDescription()
}

func TestTCPForward(t *testing.T) {
	t.Parallel()

	// 启动服务端、客户端
	s, err := setupServer([]string{
		"server",
		"-addr", "127.0.0.1:0",
		"-stunAddr", "127.0.0.1:0",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	cSlice, err := setupClients(clientOption{
		args: []string{
			"client",
			"-id", "id1",
			"-secret", "secret1",
			"-remote", s.GetListenerAddrPort().String(),
			"-local", "http://www.baidu.com/",
			"-remoteTimeout", "5s",
			"-useLocalAsHTTPHost",
			"-remoteSTUN", "stun:" + s.GetSTUNListenerAddrPort().String(),
			"-webrtcThreadMode",
		},
	}, clientOption{
		args: []string{
			"client",
			"-id", "id2",
			"-secret", "secret2",
			"-remote", s.GetListenerAddrPort().String(),
			"-local", "http://www.baidu.com/",
			"-logLevel", "debug",
			"-remoteTimeout", "5s",
			"-useLocalAsHTTPHost",
			"-remoteSTUN", "stun:" + s.GetSTUNListenerAddrPort().String(),
			"-tcpForwardAddr", "127.0.0.1:0",
			"-tcpForwardHostPrefix", "id1",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		for _, c := range cSlice {
			c.Close()
		}
	}()

	// 向 client2 的 tcpforward 地址发送 http 请求
	client2TCPForwardAddrPort := cSlice[1].GetTCPForwardListenerAddrPort()
	resp, err := http.Get("http://" + client2TCPForwardAddrPort.String() + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatal("invalid http status")
	}
	all, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) > 100 {
		all = all[:100]
	}
	t.Logf("%s", all)
}
