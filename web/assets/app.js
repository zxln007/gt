/* G-Tunnel 门户交互:压测数据渲染 / 下载联动 / 命令生成 / 数字滚动 / 滚动显现 */
(function () {
  'use strict';

  var reduceMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;

  /* ── 压测数据(来源:core/README_CN.md wrk 实测) ── */
  var SCENARIOS = {
    ubuntu: {
      rows: [
        { name: 'G-Tunnel TCP', rps: 241344.46, lat: '558.51μs', tr: '197.70MB/s', gt: true },
        { name: 'G-Tunnel QUIC', rps: 128380.49, lat: '826.65μs', tr: '105.16MB/s', gt: true },
        { name: 'frp v0.52.1', rps: 40003.03, lat: '4.49ms', tr: '31.82MB/s' }
      ],
      note: 'GT-QUIC 于 443 端口启用 QUIC 并携带证书,吞吐仍为 frp 的 3.2 倍;GT-TCP 平均延迟 558.51μs,约为 frp 的 1/8。',
      log: [
        '$ wrk -c 100 -d 30s -t 10 http://id1.example.com:12080',
        'Running 30s test @ http://id1.example.com:12080',
        '  10 threads and 100 connections',
        '  Thread Stats   Avg      Stdev     Max   +/- Stdev',
        '    Latency     558.51us    2.05ms  71.54ms   99.03%',
        '    Req/Sec     24.29k      2.28k   49.07k    95.74%',
        '  7264421 requests in 30.10s, 5.81GB read',
        'Requests/sec:  241344.46',
        'Transfer/sec:    197.70MB'
      ].join('\n')
    },
    mac: {
      rows: [
        { name: 'G-Tunnel', rps: 45811.08, lat: '2.22ms', tr: '37.14MB/s', gt: true },
        { name: 'frp dev 42745a3', rps: 1511.10, lat: '76.92ms', tr: '1.05MB/s' }
      ],
      note: '相同参数下 frp 出现 20,610 个非 2xx/3xx 响应;G-Tunnel 无一例错误,客户端常驻内存 17.8MB。',
      log: [
        '$ wrk -c 100 -d 30s -t 10 http://pi.example.com:7001   # G-Tunnel',
        'Requests/sec:  45811.08',
        'Transfer/sec:     37.14MB',
        '',
        '$ wrk -c 100 -d 30s -t 10 http://pi.example.com:7000   # frp',
        '  45487 requests in 30.10s, 31.65MB read',
        '  Non-2xx or 3xx responses: 20610',
        'Requests/sec:   1511.10',
        'Transfer/sec:      1.05MB'
      ].join('\n')
    },
    short: {
      rows: [
        { name: 'G-Tunnel QUIC', rps: 92622.63, lat: '1.84ms', tr: '11.39MB/s', gt: true },
        { name: 'G-Tunnel TCP', rps: 51822.69, lat: '4.55ms', tr: '6.38MB/s', gt: true },
        { name: 'frp v0.52.1', rps: 41334.52, lat: '2.95ms', tr: '5.09MB/s' }
      ],
      note: '响应体小于 10 字节的高频短连接:GT-QUIC 借 0-RTT 将平均延迟压至 1.84ms;GT-TCP 吞吐高于 frp 但延迟略高,可按场景选型。',
      log: [
        '$ wrk -c 100 -d 30s -t 10 http://id1.example.com:12080/',
        '  # GT-QUIC / GT-TCP / frp 平均延迟:',
        '  1.84ms / 4.55ms / 2.95ms',
        'Requests/sec:  92622.63   # GT-QUIC',
        'Requests/sec:  51822.69   # GT-TCP',
        'Requests/sec:  41334.52   # frp v0.52.1'
      ].join('\n')
    }
  };

  function fmt(n) {
    return n.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
  }

  function renderScenario(key) {
    var s = SCENARIOS[key];
    var max = Math.max.apply(null, s.rows.map(function (r) { return r.rps; }));
    var rowsEl = document.getElementById('bench-rows');
    var header = [
      '<div class="bench-header">',
      '<span class="bh-name">实现</span><span></span>',
      '<span>Req/s</span><span>平均延迟</span><span class="bh-tr">带宽</span>',
      '</div>'
    ].join('');
    var rows = s.rows.map(function (r) {
      var pct = Math.max(1, Math.round(r.rps / max * 100));
      return [
        '<div class="bench-row ', r.gt ? 'gt' : 'frp', '">',
        '<span class="br-name">', r.name, '</span>',
        '<span class="br-bar" style="width:', pct, '%"></span>',
        '<span class="br-rps num">', fmt(r.rps), '</span>',
        '<span class="br-lat num">', r.lat, '</span>',
        '<span class="br-tr num">', r.tr, '</span>',
        '</div>'
      ].join('');
    }).join('');
    rowsEl.innerHTML = header + rows;
    document.getElementById('bench-note').textContent = s.note;
    document.getElementById('bench-log').textContent = s.log;
  }

  var tabs = document.querySelectorAll('.tab');
  tabs.forEach(function (t) {
    t.addEventListener('click', function () {
      tabs.forEach(function (o) { o.setAttribute('aria-selected', String(o === t)); });
      document.getElementById('bench-body').setAttribute('aria-labelledby', t.id);
      renderScenario(t.dataset.scenario);
    });
  });
  renderScenario('ubuntu');

  /* ── 下载联动 ── */
  ['win', 'mac', 'linux'].forEach(function (os) {
    var sel = document.getElementById(os + '-arch');
    sel.addEventListener('change', function () {
      var opt = sel.options[sel.selectedIndex];
      document.getElementById(os + '-btn').href = '/downloads/' + opt.dataset.file;
      document.getElementById(os + '-file').textContent = opt.dataset.file;
    });
  });

  /* ── 命令生成器 ── */
  var inputs = ['cfg-remote', 'cfg-local', 'cfg-id', 'cfg-secret'].map(function (id) {
    return document.getElementById(id);
  });
  var cmdEl = document.getElementById('cmd-client');

  function esc(s) {
    return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
  }

  function updateCmd() {
    var v = inputs.map(function (i) { return i.value.trim(); });
    cmdEl.textContent = './gt client -local ' + (v[1] || 'http://127.0.0.1:80') +
      ' -remote ' + (v[0] || 'tcp://id1.example.com:12080') +
      ' -id ' + (v[2] || 'pi') + ' -secret ' + (v[3] || 'secret1');
  }
  inputs.forEach(function (i) { i.addEventListener('input', updateCmd); });

  var copyBtn = document.getElementById('btn-copy');
  copyBtn.addEventListener('click', function () {
    var done = function () {
      copyBtn.textContent = '已复制';
      copyBtn.classList.add('btn-primary');
      copyBtn.classList.remove('btn-ghost');
      setTimeout(function () {
        copyBtn.textContent = '复制';
        copyBtn.classList.add('btn-ghost');
        copyBtn.classList.remove('btn-primary');
      }, 2000);
    };
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(cmdEl.textContent).then(done).catch(function () {});
    } else {
      var ta = document.createElement('textarea');
      ta.value = cmdEl.textContent;
      document.body.appendChild(ta);
      ta.select();
      try { document.execCommand('copy'); done(); } catch (e) {}
      document.body.removeChild(ta);
    }
  });

  /* ── Hero 数字滚动 ── */
  function countUp(el) {
    var target = parseFloat(el.dataset.count);
    var decimals = (el.textContent.split('.')[1] || '').length;
    if (reduceMotion || !target) return;
    var dur = 1200, t0 = null;
    function frame(t) {
      if (!t0) t0 = t;
      var p = Math.min(1, (t - t0) / dur);
      var eased = 1 - Math.pow(1 - p, 3);
      el.textContent = (target * eased).toLocaleString('en-US', {
        minimumFractionDigits: decimals, maximumFractionDigits: decimals
      });
      if (p < 1) requestAnimationFrame(frame);
    }
    requestAnimationFrame(frame);
  }
  document.querySelectorAll('[data-count]').forEach(countUp);

  /* ── 滚动显现 ── */
  if (reduceMotion || !('IntersectionObserver' in window)) {
    document.querySelectorAll('.reveal').forEach(function (el) { el.classList.add('in'); });
  } else {
    var io = new IntersectionObserver(function (entries) {
      entries.forEach(function (e) {
        if (e.isIntersecting) { e.target.classList.add('in'); io.unobserve(e.target); }
      });
    }, { threshold: 0.12, rootMargin: '0px 0px -40px 0px' });
    document.querySelectorAll('.reveal').forEach(function (el) { io.observe(el); });
  }
})();
