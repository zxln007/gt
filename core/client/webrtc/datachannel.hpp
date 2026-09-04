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

#ifndef DATACHANNEL_HPP
#define DATACHANNEL_HPP

#include <api/data_channel_interface.h>

class DataChannelObserver : public webrtc::DataChannelObserver {
  public:
    webrtc::DataChannelInterface *dataChannel;

    DataChannelObserver(webrtc::DataChannelInterface *dataChannel, void *userData);
    ~DataChannelObserver();

    // 补发当前状态:观察者注册被异步投递到 network 线程,注册生效前发生的
    // open 状态变化会被吞掉(本地创建通道在 transport 已就绪时立即 open)
    void ReplayStateChange() { OnStateChange(); }

  protected:
    void OnStateChange();
    void OnMessage(const webrtc::DataBuffer &buffer);
    void OnBufferedAmountChange(uint64_t sent_data_size);

  private:
    void *userData;
};

#endif /* DATACHANNEL_HPP */
