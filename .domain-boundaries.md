# 领域边界

## 0. 目录

- [1. 项目](#1-项目)
- [2. 全局规则](#2-全局规则)
- [3. 限界上下文](#3-限界上下文)
  - [3.1 核心应用服务与搜索编排](#31-核心应用服务与搜索编排)
  - [3.2 插件注册与管理](#32-插件注册与管理)
  - [3.3 插件运行时与宿主 API](#33-插件运行时与宿主-api)
  - [3.4 插件作者契约](#34-插件作者契约)
  - [3.5 内置插件](#35-内置插件)
  - [3.6 前端启动器 UI](#36-前端启动器-ui)
  - [3.7 桌面集成](#37-桌面集成)
  - [3.8 文档与示例](#38-文档与示例)
- [4. 共享区域](#4-共享区域)
- [5. 协调规则](#5-协调规则)
- [6. 待确认问题](#6-待确认问题)
- [7. 变更记录](#7-变更记录)

## 1. 项目

- Block: 1
- 名称: RA
- 最后更新: 2026-05-22
- 来源: 混合
- 概要: 面向 Linux 的 Wails v3 启动器原型，负责桌面应用搜索、本地 WASM 插件加载、内置插件能力以及 Svelte 前端壳层。

## 2. 全局规则

- Block: 2
- 优先复用当前边界，而不是每次重新推断。
- 不要把存在所有权重叠的范围分配为并行写任务。
- 生成物和构建产物默认不进入普通实现任务，除非任务明确要求生成、打包或发布。
- `pkg/raplugin` 是插件公开契约；这里的变更必须同步考虑运行时、注册表、内置插件、示例和文档。
- 不要回退用户或其他代理的改动；发现冲突时先停下并上报。

## 3. 限界上下文

### 3.1 核心应用服务与搜索编排

- Block: 3.1
- 目标: 负责启动器主服务编排、桌面条目加载后的服务组织、搜索结果整形、动作执行、插件资源对外服务，以及暴露给 Wails 前端的应用服务接口。
- 不负责: 插件包底层校验、WASM 执行细节、插件管理持久化策略、插件自身搜索逻辑、前端展示细节。
- 主要路径:
  - `internal/app/`
  - `main.go`
  - `window_options_test.go`
- 关键模块:
  - `LauncherService`
  - `ActionExecutor`
  - 插件配置与 capability 启停编排
- 对外接口:
  - Wails 绑定的服务方法
  - `app.Result`
  - `app.Action`
  - `app.Status`
  - `app.PluginActionRequest`
- 上游依赖:
  - 3.2 插件注册与管理
  - 3.3 插件运行时与宿主 API
  - 3.7 桌面集成
- 下游依赖方:
  - 3.6 前端启动器 UI
  - 3.5 内置插件
- 并行工作规则:
  - 可安全读取: yes
  - 可安全写入: no
  - 备注: 这里承载跨上下文编排，除测试或单个局部服务方法外，默认串行写入。

### 3.2 插件注册与管理

- Block: 3.2
- 目标: 负责插件发现、注册表加载、manifest 与 capability 校验、内置/用户插件来源规则、安装卸载启停持久化，以及加载错误汇总。
- 不负责: WASM ABI 执行实现、插件作者 Go API、前端管理器界面、具体插件业务逻辑。
- 主要路径:
  - `internal/plugins/`
  - `internal/app/plugin_management.go`
  - `internal/app/pluginstore.go`
- 关键模块:
  - `plugins.Registry`
  - `plugins.Plugin`
  - 插件管理服务操作
- 对外接口:
  - 提供给 `internal/app` 的注册表加载与搜索接口
  - 提供给 `ra-plugin-manager` 与前端的插件管理接口
- 上游依赖:
  - 3.3 插件运行时与宿主 API
  - 3.4 插件作者契约
- 下游依赖方:
  - 3.1 核心应用服务与搜索编排
  - 3.5 内置插件
  - 3.6 前端启动器 UI
- 并行工作规则:
  - 可安全读取: yes
  - 可安全写入: no
  - 备注: 变更管理 API 形状、配置语义或存储规则时，必须和 3.1 协调。

### 3.3 插件运行时与宿主 API

- Block: 3.3
- 目标: 负责 WASI/WASM 编译加载、插件导出数据提取、搜索调用、超时与并发控制，以及带权限约束的宿主 API 调用。
- 不负责: 注册表策略、UI 资源路由策略、插件管理流程、前端桥接布局。
- 主要路径:
  - `internal/pluginruntime/`
- 关键模块:
  - `pluginruntime.Runtime`
  - `pluginruntime.HostAPI`
  - WASM host imports
- 对外接口:
  - `Compile`
  - `Load`
  - `LoadFromRuntime`
  - `Search`
  - 供插件调用的运行时宿主 API
- 上游依赖:
  - 3.4 插件作者契约
- 下游依赖方:
  - 3.2 插件注册与管理
  - 3.1 核心应用服务与搜索编排
- 并行工作规则:
  - 可安全读取: yes
  - 可安全写入: no
  - 备注: ABI 或 host API 变化必须同步 3.4 和 3.5，默认串行处理。

### 3.4 插件作者契约

- Block: 3.4
- 目标: 负责插件作者使用的 Go 包、WASM 导出约定、类型定义和宿主 API stub。
- 不负责: RA 服务实现、注册表存储、前端 UI、某个内置插件的产品行为。
- 主要路径:
  - `pkg/raplugin/`
- 关键模块:
  - `Manifest`
  - `Capability`
  - `SearchRequest`
  - `SearchResult`
  - WASM exports 与 host API stubs
- 对外接口:
  - `github.com/nzlov/ra/pkg/raplugin`
  - 被 `internal/pluginruntime` 消费的导出符号
- 上游依赖:
  - none
- 下游依赖方:
  - 3.3 插件运行时与宿主 API
  - 3.2 插件注册与管理
  - 3.5 内置插件
  - 3.8 文档与示例
- 并行工作规则:
  - 可安全读取: yes
  - 可安全写入: no
  - 备注: 这是跨上下文公开契约，写入必须串行，并同步更新测试、示例和文档。

### 3.5 内置插件

- Block: 3.5
- 目标: 负责内置插件源码包，以及各插件自己拥有的 capability、搜索逻辑、UI 资源和嵌入资产。
- 不负责: 核心服务管理规则、注册表校验、运行时 ABI、前端启动器外壳。
- 主要路径:
  - `plugins/`
- 关键模块:
  - `ra-app-launcher`
  - `ra-calculator`
  - `ra-plugin-manager`
  - `plugins/builtins.go`
- 对外接口:
  - 通过 `pkg/raplugin` 暴露的插件 manifest、capability、assets 与 search
  - 内置插件列表 `plugins.List`
- 上游依赖:
  - 3.4 插件作者契约
  - 3.1 核心应用服务与搜索编排
  - 3.2 插件注册与管理
- 下游依赖方:
  - 3.1 核心应用服务与搜索编排
  - 3.6 前端启动器 UI
- 并行工作规则:
  - 可安全读取: yes
  - 可安全写入: yes
  - 备注: 按单个插件目录拆分时可并行；一旦涉及 `plugins/builtins.go` 或生成产物则回到串行。

### 3.6 前端启动器 UI

- Block: 3.6
- 目标: 负责 Svelte 启动器界面、前端搜索调度、窗口行为、插件 iframe 容器、Wails 绑定消费，以及前端测试和构建配置。
- 不负责: 服务端行为、插件注册表策略、运行时权限检查、插件包契约本身。
- 主要路径:
  - `frontend/src/`
  - `frontend/tests/`
  - `frontend/index.html`
  - `frontend/package.json`
  - `frontend/vite.config.ts`
  - `frontend/svelte.config.js`
- 关键模块:
  - `frontend/src/App.svelte`
  - `frontend/src/searchScheduler.js`
  - `frontend/src/launcherWindowBehavior.js`
- 对外接口:
  - 前端消费的 Wails 生成绑定
  - capability 页面使用的 `window.ra` bridge
- 上游依赖:
  - 3.1 核心应用服务与搜索编排
  - 3.2 插件注册与管理
- 下游依赖方:
  - none
- 并行工作规则:
  - 可安全读取: yes
  - 可安全写入: yes
  - 备注: 不要手改 `frontend/bindings/` 生成内容，除非任务明确要求修改生成结果。

### 3.7 桌面集成

- Block: 3.7
- 目标: 负责 Linux `.desktop` 条目解析、默认应用目录、启动数据以及桌面/环境集成辅助逻辑。
- 不负责: 插件搜索排序、前端布局、注册表策略、插件管理界面。
- 主要路径:
  - `internal/desktop/`
  - `webkit_env.go`
  - `webkit_env_test.go`
- 关键模块:
  - `desktop.Entry`
  - `desktop.LoadDirs`
- 对外接口:
  - 提供给 `internal/app` 的桌面条目和默认目录 API
- 上游依赖:
  - Linux desktop files
  - 运行环境变量
- 下游依赖方:
  - 3.1 核心应用服务与搜索编排
  - 3.5 内置插件
- 并行工作规则:
  - 可安全读取: yes
  - 可安全写入: yes
  - 备注: 如果变更应用条目字段或启动语义，需要和 3.1 协调。

### 3.8 文档与示例

- Block: 3.8
- 目标: 负责用户文档、插件契约说明、示例插件、开发说明和计划/规格记录。
- 不负责: 当代码与文档冲突时，不负责决定运行时真实行为。
- 主要路径:
  - `README.md`
  - `README.zh-CN.md`
  - `docs/`
  - `examples/`
  - `plugins/README.md`
- 关键模块:
  - 插件文档
  - 示例插件包
- 对外接口:
  - 面向开发者和插件作者的命令、约束和接入说明
- 上游依赖:
  - 所有实现上下文
- 下游依赖方:
  - 开发者
  - 插件作者
- 并行工作规则:
  - 可安全读取: yes
  - 可安全写入: yes
  - 备注: 只改文档通常可并行，但不能先于实现擅自声明未验证的行为。

## 4. 共享区域

- Block: 4
- 路径: `pkg/raplugin/`
- 风险: high
- 规则: 插件公开契约；修改前要同步考虑运行时、注册表、内置插件、示例与文档。
- 路径: `internal/app/`
- 风险: high
- 规则: 中央编排层；除明确不重叠的局部范围外，避免并行写入。
- 路径: `internal/plugins/`
- 风险: medium
- 规则: 注册表行为会影响服务层与插件管理器，行为变更要先确认接口边界。
- 路径: `internal/pluginruntime/`
- 风险: high
- 规则: 运行时/ABI 变化必须和 `pkg/raplugin`、内置插件协同演进。
- 路径: `frontend/bindings/`
- 风险: medium
- 规则: Wails 生成绑定；优先通过既有工具链再生成，不做手工维护。
- 路径: `Taskfile.yml`
- 风险: medium
- 规则: 共享构建与打包编排入口，不作为随手并行写入目标。
- 路径: `cmd/ra-build-plugins/`
- 风险: medium
- 规则: 构建辅助入口；涉及插件打包约定时要和 3.4、3.5 一起审视。
- 路径: `plugins/builtins_data.go`
- 风险: medium
- 规则: 生成产物；除非任务明确要求生成内置插件资源，否则不要编辑或提交。
- 路径: `bin/`
- 风险: low
- 规则: 构建输出；普通源码任务不要编辑或提交。
- 路径: `frontend/dist/`
- 风险: low
- 规则: 前端构建输出；普通前端实现不要直接编辑或提交。
- 路径: `frontend/node_modules/`
- 风险: low
- 规则: 依赖目录；不要把这里当作业务代码边界的一部分。

## 5. 协调规则

- Block: 5
- 按限界上下文拆分任务，而不是按文件数量拆分。
- 跨上下文重构默认串行执行。
- 写入共享区域前必须先确认所有权是否单一明确。
- 内置插件任务尽量按单个插件目录分配。
- 涉及插件管理、权限、结果动作、UI 资源路由或公开契约时，默认视为跨上下文任务。
- 验证方式要跟随触达上下文：后端/运行时走 Go tests，前端走 npm tests/build，只有绑定或桌面打包受影响时才跑 Wails 生成或构建。

## 6. 待确认问题

- Block: 6
- 暂无。

## 7. 变更记录

- Block: 7
- 2026-05-21 初始化 RA 仓库级 DDD 边界图。
- 2026-05-21 明确插件管理持久化属于 3.2，而不是 3.1。
- 2026-05-21 补充 `Taskfile.yml`、中文 README 和动作执行相关边界。
- 2026-05-22 将边界文件统一为中文结构，并按当前仓库路径补充 `pluginstore`、`cmd/ra-build-plugins`、`frontend/node_modules` 等边界说明。
