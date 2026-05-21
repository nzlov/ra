# RA JSON Editor And Calc Paper Design

## Goal

在 RA 里按现有插件边界复刻两个 uTools 插件的核心体验：

- 将内置 `ra-calculator` 升级为“计算稿纸”能力，保留 `=` 搜索触发并支持持久化稿纸编辑。
- 新增示例插件 `ra-json-editor`，提供 JSON 文本编辑、格式化、校验和树/文本视图切换。

同时为插件新增一层最小可用的宿主私有存储 API，落盘到 SQLite，但不把数据库能力直接暴露给插件。

## Scope

### Included

- 内置插件 `ra-calculator` 升级为稿纸式计算器。
- 示例插件 `examples/ra-json-editor`。
- 插件私有存储 host API：
  - `store.get`
  - `store.set`
  - `store.delete`
  - `store.list`
- 宿主 SQLite 存储实现。
- 与上述能力直接相关的 runtime、service、docs、tests 更新。

### Excluded

- 本地 `.json` 文件读取。
- JSON Schema、JMESPath、JSONPath、XML/YAML/Base64 等扩展工具能力。
- 跨插件共享数据。
- 任意 SQL、事务、订阅、同步推送。
- 计算稿纸的高级公式系统、隐式变量引用、标签/主题系统、多窗口同步。

## Architecture

### Overview

本次改动跨越三个上下文：

- `3.3 Plugin Runtime And Host API`
- `3.4 Plugin Author Contract`
- `3.5 Built-In Plugins`

整体原则：

- 插件继续通过 Go/WASM 和嵌入式前端资源交付。
- 新增的宿主持久化能力保持最小、私有、按 `pluginID` 隔离。
- 不为两个插件做宿主特判逻辑。
- `ra-calculator` 直接升级，不拆出新的 built-in 插件。

## Plugin Storage Design

### Plugin-Facing API

插件侧新增最小文档型 KV 接口：

- `StoreGet(key string, target any) (bool, error)`
- `StoreSet(key string, value any) error`
- `StoreDelete(key string) error`
- `StoreList(prefix string, target any) error`

约束：

- `key` 由插件自己命名，例如 `papers/current`、`papers/by-id/<id>`、`drafts/latest`。
- `value` 必须是 JSON 可序列化值。
- 插件永远不拿到数据库连接、文件路径或 SQL 能力。

### Host Runtime Boundary

runtime 在处理 host call 时总是绑定当前执行插件的 `pluginID`。

真实存储维度是：

- `(plugin_id, key) -> value_json`

插件请求中不允许自带或覆盖 `pluginID`。即使插件伪造别的插件 ID，runtime 也按当前执行上下文写入和读取。

### SQLite Storage

落盘路径：

- `~/.config/ra/plugin-store.db`

首版只使用一张表：

```sql
CREATE TABLE plugin_store (
  plugin_id TEXT NOT NULL,
  key TEXT NOT NULL,
  value_json TEXT NOT NULL,
  updated_at INTEGER NOT NULL,
  PRIMARY KEY (plugin_id, key)
);
```

查询模型：

- `get`：按主键读取一条。
- `set`：按主键 upsert。
- `delete`：按主键删除。
- `list(prefix)`：`WHERE plugin_id = ? AND key LIKE ? ORDER BY key`

为什么选 SQLite：

- 比单文件 JSON 更稳定。
- 对后续数据增长更稳妥。
- 不改变插件 API，后续仍可替换宿主实现。

## Calc Paper Design

### Entry Behavior

- 保留 `=...` 搜索触发。
- 搜索结果仍打开 `ra-calculator.calculate` capability。
- 如果搜索查询里带表达式，例如 `=12*8`：
  - capability 打开后自动把表达式插入当前稿纸的新行。
- 如果只是从 capability 打开：
  - 恢复当前稿纸。

### UI Structure

- 左侧稿纸列表：
  - 标题
  - 最近更新时间
  - 新建
  - 重命名
  - 删除
- 右侧稿纸内容：
  - 多行表达式输入
  - 每行实时结果
  - 行级错误显示
- 工具区：
  - 新建稿纸
  - 清空当前稿纸
  - 复制某行结果

首版不做：

- 锁定只读
- 标签
- 主题
- 隐式变量引用
- 多窗口同步

### Data Model

插件内部使用两类记录：

- `papers/current`
- `papers/order`
- `papers/by-id/<paper-id>`

单张稿纸文档：

```json
{
  "id": "paper-1740000000000",
  "title": "2026-05-21 计算",
  "createdAt": 1740000000000,
  "updatedAt": 1740000000000,
  "lines": [
    {
      "id": "line-1",
      "expression": "12*8",
      "result": "96"
    }
  ]
}
```

行为规则：

- 首次打开时自动创建空稿纸。
- 编辑时防抖保存。
- 删除当前稿纸后自动切到最近一张；如果没有，则新建空稿纸。
- 表达式计算失败只影响当前行显示，不阻断继续编辑。

## JSON Editor Design

### Delivery Shape

- 新建示例插件：`examples/ra-json-editor`
- 一个 capability：
  - `json-editor`

### Triggers

- 关键词触发：
  - `json`
  - `json edit`
  - `json format`
- 内容触发：
  - 当查询文本看起来像完整 JSON 对象或数组时命中。
- 打开时：
  - 若查询里带 JSON 文本，则自动灌入编辑器。
  - 否则进入空编辑器。

### UI Structure

- 顶部工具栏：
  - 格式化
  - 压缩
  - 校验
  - 文本/树形切换
  - 清空
- 主编辑区：
  - 文本模式
  - 树形模式
- 底部状态区：
  - 校验结果
  - 错误位置
  - 当前模式

### Capability Boundary

首版要做：

- JSON 文本输入
- 格式化
- 压缩为单行
- 语法校验
- 文本视图
- 树形视图
- 查询文本自动带入

首版不做：

- 本地文件读取
- JSON Schema
- JMESPath / JSONPath
- XML / YAML / Base64 转换
- 多草稿管理

### Optional Persistence

可以顺手保存：

- `drafts/latest`
- `ui/view-mode`

但不是首版硬要求。功能优先级低于编辑、格式化和校验。

## Error Handling

### Storage API

- 插件传入不可序列化值：返回明确错误。
- SQLite 打开失败或写入失败：宿主返回错误给插件，插件 UI 显示可恢复提示。
- 不存在 key：
  - `get` 返回 `found=false`
  - `list` 返回空集合

### Calc Paper

- 非法表达式：
  - 当前行显示错误状态
  - 不阻断其他行编辑或保存
- 稿纸保存失败：
  - UI 显示保存失败状态
  - 保留内存中的编辑内容，允许再次尝试

### JSON Editor

- 非法 JSON：
  - 校验区显示错误信息和位置
  - 文本仍可继续编辑
- 树形视图切换失败：
  - 保持文本视图，不清空原始内容

## Testing Strategy

### Runtime And Contract

- 插件存储 host API 的单元测试：
  - `get/set/delete/list`
  - 插件隔离
  - 非法 JSON 值
- WASM runtime 集成测试：
  - 真实测试插件通过 host API 读写 SQLite 存储

### Service And Registry

- `ra-calculator` 搜索仍命中 `=...`
- capability 打开时查询参数透传
- 新示例插件能被 registry 加载并参与搜索

### Plugin UI

- `ra-calculator` 前端脚本测试：
  - 打开时注入表达式
  - 计算结果更新
  - 空稿纸创建
- `ra-json-editor` 前端测试：
  - 格式化
  - 压缩
  - 非法 JSON 校验
  - 文本/树形切换

## Files And Ownership

### Runtime / Contract

- `pkg/raplugin/`
- `internal/pluginruntime/`
- `internal/app/` 中与配置/路径/服务初始化直接相关部分

### Built-In Plugin

- `plugins/ra-calculator/`

### Example Plugin

- `examples/ra-json-editor/`

### Docs

- `docs/plugins.md`
- `README.md`
- `README.zh-CN.md`

## Open Decisions Resolved

- `计算稿纸`：内置插件，保留 `=` 搜索触发并增强。
- `JSON 编辑器`：示例插件，不做本地文件读取。
- 插件存储：宿主私有 API，落盘启用 SQLite3。

## Change Impact Summary

这次不是单纯增加两个 UI，而是补齐一层可复用的插件私有持久化能力。范围仍然受控：

- 不改变 RA 的插件身份模型。
- 不引入跨插件共享或任意文件访问。
- 不让宿主为了两个插件出现专有分支逻辑。

因此这次改动是“扩一层通用最小基础设施，再用它实现两个具体插件”。
