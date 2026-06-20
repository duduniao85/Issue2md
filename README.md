# issue2md

> 将 GitHub Issue / Pull Request / Discussion 转换为带 YAML front matter 的本地 Markdown。
> 零第三方依赖，纯 Go 标准库实现（含图片本地化）。

## 安装

```bash
make build      # 产出 bin/issue2md
```

## 用法

```bash
issue2md [flags] <github-url>
```

### Flags
| Flag | 默认 | 说明 |
| --- | --- | --- |
| `-o` | 自动命名 | 输出文件路径；`-o -` 输出到 stdout |
| `-token` | `$GITHUB_TOKEN` | GitHub Token（私有仓库必需） |
| `-no-images` | false | 不下载图片，保留远程链接 |
| `-timeout` | 30s | 单次 HTTP 请求超时 |
| `-baseurl` | https://api.github.com | GitHub API 根（高级/测试钩子） |
| `-v` | — | 打印版本并退出 |

### 示例
```bash
# 公开 Issue
issue2md https://github.com/golang/go/issues/123

# 私有仓库 PR（依赖 $GITHUB_TOKEN）
GITHUB_TOKEN=ghp_xxx issue2md https://github.com/acme/internal/pull/42

# 管道
issue2md -o - https://github.com/golang/go/issues/123 | grep -i bug

# 不下载图片
issue2md -no-images https://github.com/golang/go/discussions/7
```

### 退出码
| 码 | 含义 |
| --- | --- |
| 0 | 成功 |
| 1 | 本地 IO 错误（写文件/目录） |
| 2 | 使用错误（参数/URL 非法） |
| 3 | GitHub API 错误（404/401/速率/5xx/网络） |

## 输出
- 默认文件：`{owner}-{repo}-{type}-{number}.md`（当前目录）
- 图片目录：`{owner}-{repo}-{type}-{number}.files/`（链接改写为相对路径）
- stdout 模式（`-o -`）：不下载图片，输出含原始远程链接

## Token
- 匿名访问公开仓库（受 60 次/小时速率限制）
- 设置 `GITHUB_TOKEN` 访问私有仓库（5000 次/小时）

## 开发
```bash
make test        # 全量测试
make vet         # 静态检查
make build       # 构建
```

### 设计文档
| 文档 | 作用 |
| --- | --- |
| [`constitution.md`](./constitution.md) | 项目宪法 |
| [`spec.md`](./spec.md) | 需求规格 |
| [`API-sketch.md`](./API-sketch.md) | 接口设计 |
| [`specs/001-core-functionality/plan.md`](./specs/001-core-functionality/plan.md) | 技术方案 |
| [`specs/001-core-functionality/tasks.md`](./specs/001-core-functionality/tasks.md) | 任务分解 |

## 开发约定（宪法摘要）
- 测试先行（TDD + 表格驱动 + httptest 拒绝 Mock）
- 错误 `fmt.Errorf("...: %w", err)` 链式包装，支持 `errors.Is/As`
- 无全局可变状态，依赖注入
- 标准库优先，零第三方依赖
