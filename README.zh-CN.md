# RA

English documentation: [README.md](README.md)

RA 是一个基于 Go 和 Wails v3 的 Linux 优先启动器原型，目标是提供类似 uTools 的轻量工作流：搜索应用并打开本地插件能力。

## 当前 MVP

- 扫描 `/usr/share/applications` 和 `~/.local/share/applications` 下的 `.desktop` 文件。
- 通过内置 `ra-app-launcher` 插件提供应用搜索和启动能力。
- 通过内置 `ra-calculator` 插件支持计算器查询，例如 `=6*7`。
- 从仓库内 `plugins/` 目录加载内置插件源码。
- 从 `~/.local/share/ra/plugins/*.wasm` 加载用户插件包。
- 提供内置 `ra-plugin-manager` 插件，用于本地插件安装、启用、禁用、卸载和刷新。
- 将插件建模为 Go/WASI `.wasm` 文件，包含插件自有源码、manifest、capability、权限、搜索行为和嵌入式 UI 资源。
- 支持 capability 级别的启用和禁用。
- 通过沙箱 iframe 在 `/plugins/<plugin-id>/<capability-id>/...` 下提供已启用 capability 的 UI 资源。
- 向插件暴露带权限校验的 RA API，包括 `apps.list` 这类 WASM Host API，以及通过 `window.ra.invoke()` 触发的 UI 操作。

## 依赖

- Go 1.25+
- Wails v3 alpha CLI，可通过 `go install github.com/wailsapp/wails/v3/cmd/wails3@latest` 安装
- Node.js 和 npm
- Wails 在 Linux 下需要的 GTK4/WebKitGTK 6.0 桌面依赖

在 CachyOS/Arch 上，相关包为 `base-devel`、`gtk4` 和 `webkitgtk-6.0`。

## 开发

```sh
cd frontend
npm install
cd ..
env GOCACHE=/tmp/ra-go-cache CGO_ENABLED=1 go test ./... -count=1
env GOCACHE=/tmp/ra-go-cache CGO_ENABLED=1 wails3 task dev
```

如果只想单独启动前端开发服务器：

```sh
wails3 task dev:frontend
```

如果你不想走 Wails task 封装，也可以手动执行：

```sh
cd frontend
npm run build
cd ..
env GOCACHE=/tmp/ra-go-cache CGO_ENABLED=1 wails3 generate bindings -f '-gcflags=all=\"-l\"' -ts
env GOCACHE=/tmp/ra-go-cache CGO_ENABLED=1 wails3 dev -config ./build/config.yml -port 9245
```

## 构建

为当前操作系统构建：

```sh
env GOCACHE=/tmp/ra-go-cache CGO_ENABLED=1 wails3 task build
```

常见 Linux 构建命令：

```sh
env GOCACHE=/tmp/ra-go-cache CGO_ENABLED=1 wails3 task build:linux:debug:amd64
env GOCACHE=/tmp/ra-go-cache CGO_ENABLED=1 wails3 task build:linux:prod:amd64
```

在 Linux 下，构建出的二进制位于 `bin/ra`。

## 打包

为当前操作系统打包：

```sh
env GOCACHE=/tmp/ra-go-cache CGO_ENABLED=1 wails3 task package
```

Linux AppImage 打包：

```sh
env GOCACHE=/tmp/ra-go-cache CGO_ENABLED=1 wails3 task package:linux
```

当前这台机器的 Go 环境默认带有 `CGO_ENABLED=0`。Wails 在 Linux 下依赖 WebKitGTK，因此测试、构建和打包桌面目标时需要显式设置 `CGO_ENABLED=1`。

在 Linux 下，打包产物会输出到 `bin/` 目录。任务文件中也包含 macOS `.app` 打包和 Windows NSIS 安装包打包。

## 运行

运行当前操作系统对应的已构建应用：

```sh
wails3 task run
```

在 Linux 下也可以直接运行二进制：

```sh
./bin/ra
```

## 插件格式

当前本地插件契约见 `docs/plugins.md`。

内置插件源码位于仓库 `plugins/` 目录。示例插件源码位于 `examples/`。用户安装的插件包应放在 `~/.local/share/ra/plugins/<plugin-id>.wasm`。

构建一个插件包：

```sh
GOOS=wasip1 GOARCH=wasm go build -buildvcs=false -buildmode=c-shared -o codec-tools.wasm ./examples/codec-tools
```

插件和 capability 的启用/禁用状态存放在 `~/.config/ra/plugins.json`。插件管理器可以禁用 `ra-app-launcher` 这类内置插件，但只会卸载用户插件文件，并且拒绝禁用或卸载它自己的管理 capability。

## 后续方向

- 增加更明确的存储和结果渲染 Host API。
- 增加更适合 Niri 的显示/隐藏集成，并补充 compositor 快捷键配置说明。
