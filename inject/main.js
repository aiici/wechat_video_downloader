const ALPHABET = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";

function __wx_uid__() {
  return random_string(12);
}

function random_string(length) {
  return random_string_with_alphabet(length, ALPHABET);
}

function random_string_with_alphabet(length, alphabet) {
  const result = Array.from({ length }, () =>
    alphabet[Math.floor(Math.random() * alphabet.length)]
  );
  return result.join("");
}

function sleep(ms = 1000) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function __wx_channels_copy(text) {
  const textArea = document.createElement("textarea");
  textArea.value = text;
  textArea.style.cssText = "position: fixed; top: -999px;";
  document.body.appendChild(textArea);
  textArea.select();
  document.execCommand("copy");
  document.body.removeChild(textArea);
}

function __wx_channel_loading() {
  return window.__wx_channels_tip__?.loading?.("下载中") || { hide() {} };
}

function __wx_log(msg) {
  fetch("/__wx_channels_api/tip", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(msg),
  });
}

function __wx_channels_video_decrypt(buffer, offset, profile) {
  const r = new Uint8Array(buffer);
  for (let i = 0; i < buffer.byteLength && offset + i < profile.decryptor_array.length; i++) {
    r[i] ^= profile.decryptor_array[i];
  }
  return r;
}

window.VTS_WASM_URL = "https://res.wx.qq.com/t/wx_fed/cdn_libs/res/decrypt-video-core/1.3.0/wasm_video_decode.wasm";
window.MAX_HEAP_SIZE = 33554432;
let decryptor_array;
let decryptor;
let loaded = false;

function wasm_isaac_generate(ptr, len) {
  const reversed = new Uint8Array(Module.HEAPU8.buffer, ptr, len).reverse();
  decryptor_array = new Uint8Array(len);
  decryptor_array.set(reversed);
  if (decryptor?.delete) decryptor.delete();
}

async function __wx_channels_decrypt(seed) {
  if (!loaded) {
    await __wx_load_script("https://res.wx.qq.com/t/wx_fed/cdn_libs/res/decrypt-video-core/1.3.0/wasm_video_decode.js");
    loaded = true;
  }
  await sleep();
  decryptor = new Module.WxIsaac64(seed);
  decryptor.generate(131072);
  return decryptor_array;
}

async function show_progress_or_loaded_size(response) {
  const contentLength = parseInt(response.headers.get("Content-Length") || "0", 10);
  const chunks = [];
  const reader = response.body.getReader();
  let loadedSize = 0;

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    chunks.push(value);
    loadedSize += value.length;
    __wx_log({ msg: contentLength ? `${((loadedSize / contentLength) * 100).toFixed(2)}%` : `${loadedSize} Bytes` });
  }

  return new Blob(chunks);
}

async function __wx_channels_download(profile, filename) {
  const blob = new Blob(profile.data, { type: "video/mp4" });
  await __wx_load_script("https://res.wx.qq.com/t/wx_fed/cdn_libs/res/FileSaver.min.js");
  saveAs(blob, filename + ".mp4");
}

async function __wx_channels_download2(profile, filename) {
  await __wx_load_script("https://res.wx.qq.com/t/wx_fed/cdn_libs/res/FileSaver.min.js");
  const ins = __wx_channel_loading();
  const response = await fetch(profile.url);
  const blob = await show_progress_or_loaded_size(response);
  __wx_log({ msg: "下载完成" });
  ins.hide();
  saveAs(blob, filename + ".mp4");
}

async function __wx_channels_download3(profile, filename) {
  await __wx_load_script("https://res.wx.qq.com/t/wx_fed/cdn_libs/res/FileSaver.min.js");
  const zip = new JSZip();
  zip.file("contact.txt", JSON.stringify(profile.contact, null, 2));
  const folder = zip.folder("images");
  await Promise.all(profile.files.map(async (f, i) => {
    const res = await fetch(f.url);
    const blob = await res.blob();
    folder.file(`${i + 1}.png`, blob);
  }));
  const content = await zip.generateAsync({ type: "blob" });
  saveAs(content, filename + ".zip");
}

async function __wx_channels_download4(profile, filename) {
  await __wx_load_script("https://res.wx.qq.com/t/wx_fed/cdn_libs/res/FileSaver.min.js");
  const ins = __wx_channel_loading();
  const response = await fetch(profile.url);
  const blob = await show_progress_or_loaded_size(response);
  __wx_log({ msg: "下载完成，开始解密" });
  let array = new Uint8Array(await blob.arrayBuffer());
  if (profile.decryptor_array) {
    array = __wx_channels_video_decrypt(array, 0, profile);
  }
  ins.hide();
  saveAs(new Blob([array], { type: "video/mp4" }), filename + ".mp4");
}

function __wx_load_script(src) {
  return new Promise((resolve, reject) => {
    const script = document.createElement("script");
    script.src = src;
    script.onload = resolve;
    script.onerror = reject;
    document.head.appendChild(script);
  });
}

function __wx_channels_handle_copy__() {
  __wx_channels_copy(location.href);
  window.__wx_channels_tip__?.toast?.("复制成功", 1000);
}

async function __wx_channels_handle_log__() {
  await __wx_load_script("https://res.wx.qq.com/t/wx_fed/cdn_libs/res/FileSaver.min.js");
  const blob = new Blob([document.body.innerHTML], { type: "text/plain;charset=utf-8" });
  saveAs(blob, "log.txt");
}

async function __wx_channels_handle_click_download__(spec) {
  const profile = window.__wx_channels_store__?.profile;
  if (!profile) return alert("检测不到视频，请将本工具更新到最新版");

  let filename = profile.title || profile.id || Date.now().toString();
  const _profile = { ...profile };

  if (spec) {
    _profile.url += `&X-snsvideoflag=${spec.fileFormat}`;
    filename += `_${spec.fileFormat}`;
  }

  __wx_log({ msg: `${filename}\n${location.href}\n${_profile.url}\n${_profile.key || ""}` });

  if (_profile.type === "picture") return __wx_channels_download3(_profile, filename);
  if (!_profile.key) return __wx_channels_download2(_profile, filename);

  _profile.data = window.__wx_channels_store__.buffers;
  try {
    _profile.decryptor_array = await __wx_channels_decrypt(_profile.key);
  } catch (e) {
    __wx_log({ msg: "解密失败，停止下载" });
    return alert("解密失败，停止下载");
  }

  __wx_channels_download4(_profile, filename);
}

function __wx_channels_download_cur__() {
  const profile = window.__wx_channels_store__?.profile;
  if (!profile) return alert("检测不到视频，请将本工具更新到最新版");
  if (!window.__wx_channels_store__.buffers?.length) return alert("没有可下载的内容");
  const filename = profile.title || profile.id || Date.now().toString();
  profile.data = window.__wx_channels_store__.buffers;
  __wx_channels_download(profile, filename);
}

async function __wx_channels_handle_download_cover() {
  const profile = window.__wx_channels_store__?.profile;
  if (!profile) return alert("检测不到视频，请将本工具更新到最新版");
  const filename = profile.title || profile.id || Date.now().toString();
  await __wx_load_script("https://res.wx.qq.com/t/wx_fed/cdn_libs/res/FileSaver.min.js");
  __wx_log({ msg: `下载封面\n${profile.coverUrl}` });
  const ins = __wx_channel_loading();
  try {
    const url = profile.coverUrl.replace(/^http/, "https");
    const res = await fetch(url);
    const blob = await res.blob();
    saveAs(blob, filename + ".jpg");
  } catch (err) {
    alert(err.message);
  }
  ins.hide();
}

// 全局变量初始化
window.__wx_channels_tip__ ||= {};
window.__wx_channels_store__ ||= {
  profile: null,
  profiles: [],
  keys: {},
  buffers: [],
};

// 注入按钮
const $icon = document.createElement("div");
$icon.innerHTML = `
  <div class="click-box op-item item-gap-combine" role="button" aria-label="下载"
    style="padding: 4px; --border-radius: 4px;">
    <svg class="svg-icon icon" viewBox="0 0 1024 1024" fill="currentColor" width="28" height="28">
      <path d="M213.333333 853.333333h597.333334v-85.333333H213.333333m597.333334-384h-170.666667V128H384v256H213.333333l298.666667 298.666667 298.666667-298.666667z"></path>
    </svg>
  </div>
`;
const __wx_channels_video_download_btn__ = $icon.firstChild;

__wx_channels_video_download_btn__.onclick = () => {
  if (window.__wx_channels_store__.profile) {
    __wx_channels_handle_click_download__(window.__wx_channels_store__.profile.spec?.[0]);
  }
};

// 注入逻辑
let count = 0;
__wx_log({ msg: "等待注入下载按钮" });

let __timer = setInterval(() => {
  count++;
  const $wrap3 = document.querySelector(".full-opr-wrp.layout-row");
  if (!$wrap3) {
    if (count >= 5) {
      clearInterval(__timer);
      __timer = null;
      __wx_log({ msg: "没有找到操作栏，注入下载按钮失败\n请在「更多」菜单中下载" });
    }
    return;
  }

  clearInterval(__timer);
  __timer = null;
  const relative = $wrap3.lastElementChild;
  if (!relative) {
    $wrap3.appendChild(__wx_channels_video_download_btn__);
    __wx_log({ msg: "注入下载按钮成功1!" });
  } else {
    $wrap3.insertBefore(__wx_channels_video_download_btn__, relative);
    __wx_log({ msg: "注入下载按钮成功2!" });
  }
}, 1000);
