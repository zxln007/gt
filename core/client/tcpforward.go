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

package client

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/isrc-cas/gt/client/std"
	"github.com/isrc-cas/gt/client/webrtc"
	"github.com/isrc-cas/gt/pool"
)

// forwardPeerConnection 包装网关侧的一条 PC。peerID 是 provider 侧既有 peer
// task 的 id（首轮交换响应尾部捕获）,后续重协商请求靠它路由到该 task。
type forwardPeerConnection struct {
	peerConnection *webrtc.PeerConnection
	peerID         uint32
	// negotiationMtx 串行化该 PC 上的重协商:多条 TCP 连接并发到来时,
	// offer/answer 必须逐个完成（信令状态机要求稳定态）
	negotiationMtx sync.Mutex
}

// renegotiate 为新建的 in-band 数据通道发起重新协商:CreateOffer → XP 请求
// （WebRTC-OP-ID 路由到 provider 的既有 task）→ 应用 answer。新版 libwebrtc
// 移除了 in-band 通道在既有 SCTP 关联上的隐式流建立,通道必须经重新协商
// 进入 SDP 才会 open。
func (f *forwardPeerConnection) renegotiate(c *Client, d dialer) (err error) {
	f.negotiationMtx.Lock()
	defer f.negotiationMtx.Unlock()

	offer, err := f.peerConnection.CreateOffer()
	if err != nil {
		return
	}
	err = f.peerConnection.SetLocalDescription(offer)
	if err != nil {
		return
	}
	offerBytes, err := json.Marshal(f.peerConnection.GetLocalDescription())
	if err != nil {
		return
	}
	conn, err := d.dial()
	if err != nil {
		return
	}
	defer func() {
		cErr := conn.Close()
		if err == nil {
			err = cErr
		}
	}()
	offerBuf := &bytes.Buffer{}
	chunkedWriter := std.NewChunkedWriter(offerBuf)
	_, err = chunkedWriter.Write(append([]byte{byte(len(offerBytes) >> 8), byte(len(offerBytes))}, offerBytes...))
	if err != nil {
		return
	}
	req, err := http.NewRequest("XP", "http://"+c.Config().TCPForwardHostPrefix+".example.com", nil)
	if err != nil {
		return
	}
	req.Header.Set("WebRTC-OP-ID", strconv.FormatUint(uint64(f.peerID), 10))
	req.Body = io.NopCloser(offerBuf)
	c.Logger.Debug().Uint32("peerTask", f.peerID).Msg("send renegotiation offer")
	resp, err := p2pSignalingHTTPClient(conn).Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		err = errors.New("invalid status code")
		return
	}

	// 重协商不产生新的 gathering（ICE 传输与凭据不变）,响应只有 answer 本身
	var answerLen uint16
	err = binary.Read(resp.Body, binary.BigEndian, &answerLen)
	if err != nil {
		return
	}
	answerJSON := make([]byte, answerLen)
	_, err = io.ReadFull(resp.Body, answerJSON)
	if err != nil {
		return
	}
	var answer webrtc.SessionDescription
	err = json.Unmarshal(answerJSON, &answer)
	if err != nil {
		return
	}
	c.Logger.Debug().Msgf("get renegotiation answer: %s\n", string(answerJSON))
	err = f.peerConnection.SetRemoteDescription(&answer)
	return
}

// p2pSignalingHTTPClient 把单条隧道连接包成只走该连接的 HTTP 客户端,
// 供 P2P 信令（XP 请求）使用。
func p2pSignalingHTTPClient(conn net.Conn) *http.Client {
	dialFn := func(ctx context.Context, network string, address string) (net.Conn, error) {
		return conn, nil
	}
	return &http.Client{
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			DialContext:           dialFn,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       5 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
}

func (c *Client) tcpForwardStart(dialer dialer) {
	// 通过函数的形式隐藏 peerConnection 并发的过程
	getPeerConnection, err := c.createPeerConnections(dialer)
	if err != nil {
		c.Logger.Error().Err(err).Msg("failed to create peer connections")
		return
	}

	// 转发 tcp 的数据
	var tempDelay time.Duration // how long to sleep on accept failure
	for {
		conn, err := c.tcpForwardListener.Accept()
		if err != nil {
			if atomic.LoadUint32(&c.closing) > 0 {
				return
			}
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				c.Logger.Error().Err(err).Dur("delay", tempDelay).Msg("Client tcp forward accept error")
				time.Sleep(tempDelay)
				continue
			}
			return
		}

		go func() {
			c.Logger.Info().Msg("tcp forward started")
			defer func() {
				c.Logger.Info().Msg("tcp forward stopped")
				_ = conn.Close()
			}()

			// 创建 in-band 数据通道（negotiated=false,id 由 libwebrtc 协商时分配）
			waitStateChange := make(chan struct{}, 1)
			var dataChannel *webrtc.DataChannel
			dataChannelConfig := webrtc.DataChannelConfig{
				OnStateChange: func(state webrtc.DataState) {
					c.Logger.Debug().Str("state", state.String()).Msg("data channel state change")
					select {
					case waitStateChange <- struct{}{}:
					default:
					}
				},
				OnMessage: func(message []byte) {
					_, err := conn.Write(message)
					if err != nil {
						c.Logger.Error().Err(err).Msg("failed to write to conn")
						return
					}
				},
			}
			fpc := getPeerConnection()
			err = fpc.peerConnection.CreateDataChannelWithID(conn.RemoteAddr().String(), 0, false, true, &dataChannelConfig, &dataChannel)
			if err != nil {
				c.Logger.Error().Err(err).Msg("failed to create data channel")
				return
			}
			defer func() {
				c.Logger.Debug().
					Int("id", dataChannel.ID).
					Str("label", dataChannel.Label).
					Str("state", dataChannel.State().String()).
					Str("error", dataChannel.Error()).
					Uint32("messageSent", dataChannel.MessageSent()).
					Uint32("messageReceived", dataChannel.MessageReceived()).
					Uint64("bytesSent", dataChannel.BytesSent()).
					Uint64("bytesReceived", dataChannel.BytesReceived()).
					Uint64("bufferedAmount", dataChannel.BufferedAmount()).
					Msg("close data channel")
				dataChannel.Close()
			}()

			// 通道必须经重新协商进入 SDP 才会 open
			err = fpc.renegotiate(c, dialer)
			if err != nil {
				c.Logger.Error().Err(err).Msg("failed to renegotiate data channel")
				return
			}
			<-waitStateChange
			buf := pool.BytesPool.Get().([]byte)
			defer pool.BytesPool.Put(buf)
			for {
				nread, err := conn.Read(buf)
				if nread > 0 {
					if !dataChannel.Send(buf[:nread]) {
						c.Logger.Error().Msg("failed to send message with data channel")
						return
					}
				}
				if err != nil {
					if err == io.EOF {
						return
					}
					c.Logger.Error().Err(err).Msg("failed to read from conn")
					return
				}
			}
		}()
	}
}

func (c *Client) createPeerConnections(dialer dialer) (getPeerConnection func() *forwardPeerConnection, err error) {
	var peerConnections []*forwardPeerConnection
	for i := uint(0); i < c.Config().TCPForwardConnections; i++ {
		var peerConnection *forwardPeerConnection
		peerConnection, err = c.createPeerConnection(dialer)
		if err != nil {
			return
		}
		peerConnections = append(peerConnections, peerConnection)
	}

	var i atomic.Uint32
	getPeerConnection = func() *forwardPeerConnection {
		iLoad := i.Load()
		iLoad %= uint32(len(peerConnections))
		i.Store(iLoad + 1)
		return peerConnections[iLoad]
	}
	return
}

func (c *Client) createPeerConnection(dialer dialer) (fpc *forwardPeerConnection, err error) {
	// 设置 peerConnection
	candidateDoneChan := make(chan struct{}, 1)
	waitNegotiationNeeded := make(chan struct{}, 1)
	peerConnectionConfig := webrtc.PeerConnectionConfig{
		ICEServers: []string{},
		OnSignalingChange: func(state webrtc.SignalingState) {
			c.Logger.Debug().Str("state", state.String()).Msg("peer connection signaling state change")
			if state == webrtc.SignalingStateClosed {
				select {
				case candidateDoneChan <- struct{}{}:
				default:
				}
				select {
				case waitNegotiationNeeded <- struct{}{}:
				default:
				}
			}
		},
		OnDataChannel: func(dataChannel *webrtc.DataChannelWithoutCallback) {
		},
		OnRenegotiationNeeded: func() {
		},
		OnNegotiationNeeded: func() {
			select {
			case waitNegotiationNeeded <- struct{}{}:
			default:
			}
		},
		OnICEConnectionChange: func(state webrtc.ICEConnectionState) {
		},
		OnStandardizedICEConnectionChange: func(state webrtc.ICEConnectionState) {
		},
		OnConnectionChange: func(state webrtc.PeerConnectionState) {
			c.Logger.Debug().Str("state", state.String()).Msg("peer connection state change")
		},
		OnICEGatheringChange: func(state webrtc.ICEGatheringState) {
			if state == webrtc.ICEGatheringStateComplete {
				select {
				case candidateDoneChan <- struct{}{}:
				default:
				}
			}
			c.Logger.Debug().Str("state", state.String()).Msg("peer connection ice gathering state change")
		},
		OnICECandidate: func(iceCandidate *webrtc.ICECandidate) {
			c.Logger.Debug().Msgf("get peer connection ice candidate: '%v'", iceCandidate)
		},
		OnICECandidateError: func(addrss string, port int, url string, errorCode int, errorText string) {
		},
	}
	signalingThread := c.webrtcThreadPool.GetThread()
	networkThread := c.webrtcThreadPool.GetSocketThread()
	workerThread := c.webrtcThreadPool.GetThread()
	var peerConnection *webrtc.PeerConnection
	err = webrtc.NewPeerConnection(&peerConnectionConfig, &peerConnection, signalingThread, networkThread, workerThread)
	if err != nil {
		return
	}
	var dataChannelUnused *webrtc.DataChannel
	err = peerConnection.CreateDataChannelWithID("only", webrtcTriggerChannelID, true, false, nil, &dataChannelUnused)
	if err != nil {
		return
	}
	c.Logger.Debug().
		Int("id", dataChannelUnused.ID).
		Str("label", dataChannelUnused.Label).
		Str("state", dataChannelUnused.State().String()).
		Str("error", dataChannelUnused.Error()).
		Uint32("messageSent", dataChannelUnused.MessageSent()).
		Uint32("messageReceived", dataChannelUnused.MessageReceived()).
		Uint64("bytesSent", dataChannelUnused.BytesSent()).
		Uint64("bytesReceived", dataChannelUnused.BytesReceived()).
		Uint64("bufferedAmount", dataChannelUnused.BufferedAmount()).
		Msg("tunnel data channel created (kept open: offer must carry the SCTP section)")

	// 发送 offer
	conn, err := dialer.dial()
	if err != nil {
		return
	}
	req, err := http.NewRequest("XP", "http://"+c.Config().TCPForwardHostPrefix+".example.com", nil)
	if err != nil {
		return
	}
	<-waitNegotiationNeeded
	offer, err := peerConnection.CreateOffer()
	if err != nil {
		return
	}
	err = peerConnection.SetLocalDescription(offer)
	if err != nil {
		return
	}
	<-candidateDoneChan
	offerBuf := &bytes.Buffer{}
	chunkedWriter := std.NewChunkedWriter(offerBuf)
	offerBytes, err := json.Marshal(peerConnection.GetLocalDescription())
	if err != nil {
		return
	}
	c.Logger.Debug().Msgf("send offer: %s\n", string(offerBytes))
	_, err = chunkedWriter.Write(append([]byte{byte(len(offerBytes) >> 8), byte(len(offerBytes))}, offerBytes...))
	if err != nil {
		return
	}
	req.Body = io.NopCloser(offerBuf)
	resp, err := p2pSignalingHTTPClient(conn).Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		err = errors.New("invalid status code")
		return
	}

	// 获取 answer、candidate 与 provider 侧 peer task id（响应尾部 {"id":N}）
	var peerID uint32
	var answerLen uint16
	err = binary.Read(resp.Body, binary.BigEndian, &answerLen)
	if err != nil {
		return
	}
	answerJSON := make([]byte, answerLen)
	_, err = io.ReadFull(resp.Body, answerJSON)
	if err != nil {
		return
	}
	var answer webrtc.SessionDescription
	err = json.Unmarshal(answerJSON, &answer)
	if err != nil {
		return
	}
	c.Logger.Debug().Msgf("get answer: %s\n", string(answerJSON))
	err = peerConnection.SetRemoteDescription(&answer)
	if err != nil {
		return
	}
	for {
		var candidateLen uint16
		err = binary.Read(resp.Body, binary.BigEndian, &candidateLen)
		if err != nil {
			if err == io.EOF {
				err = errors.New("peer id is not found in the offer response")
			}
			return
		}
		candidateJSON := make([]byte, candidateLen)
		_, err = io.ReadFull(resp.Body, candidateJSON)
		if err != nil {
			return
		}
		// {"id":N} 是响应尾部的路由信息,不是 candidate
		var probe map[string]json.RawMessage
		if json.Unmarshal(candidateJSON, &probe) == nil {
			if _, ok := probe["id"]; ok {
				var idMsg struct {
					ID uint32
				}
				err = json.Unmarshal(candidateJSON, &idMsg)
				if err != nil {
					return
				}
				peerID = idMsg.ID
				break
			}
		}
		c.Logger.Debug().Msgf("get candidate: %s\n", string(candidateJSON))
		var candidate webrtc.ICECandidate
		err = json.Unmarshal(candidateJSON, &candidate)
		if err != nil {
			return
		}
		err = peerConnection.AddICECandidate(&candidate)
		if err != nil {
			return
		}
	}
	if peerID == 0 {
		err = errors.New("invalid peer id")
		return
	}
	fpc = &forwardPeerConnection{
		peerConnection: peerConnection,
		peerID:         peerID,
	}
	return
}
