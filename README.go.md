# paddleocr-cli

OCR 命令行工具，调用 PaddleOCR AI Studio API 识别 PDF/图片，输出 Markdown 或 JSON。

## 安装

```bash
# macOS (Apple Silicon)
curl -sL https://github.com/Explorer1092/paddleocr_cli/releases/latest/download/paddleocr-cli_darwin_arm64.tar.gz | tar xz && sudo mv paddleocr-cli /usr/local/bin/

# macOS (Intel)
curl -sL https://github.com/Explorer1092/paddleocr_cli/releases/latest/download/paddleocr-cli_darwin_amd64.tar.gz | tar xz && sudo mv paddleocr-cli /usr/local/bin/

# Linux (x64)
curl -sL https://github.com/Explorer1092/paddleocr_cli/releases/latest/download/paddleocr-cli_linux_amd64.tar.gz | tar xz && sudo mv paddleocr-cli /usr/local/bin/

# Linux (ARM64)
curl -sL https://github.com/Explorer1092/paddleocr_cli/releases/latest/download/paddleocr-cli_linux_arm64.tar.gz | tar xz && sudo mv paddleocr-cli /usr/local/bin/

# Windows (PowerShell)
Invoke-WebRequest -Uri https://github.com/Explorer1092/paddleocr_cli/releases/latest/download/paddleocr-cli_windows_amd64.zip -OutFile paddleocr-cli.zip; Expand-Archive paddleocr-cli.zip -DestinationPath .; Remove-Item paddleocr-cli.zip
```

## 配置

```bash
paddleocr-cli configure --server-url URL --token TOKEN
paddleocr-cli configure --test  # 验证
```

## 使用

```bash
paddleocr-cli file.pdf              # 输出 Markdown 到 stdout
paddleocr-cli file.pdf -o out.md    # 输出到文件
paddleocr-cli file.pdf --json       # JSON 格式
```

### 参数

| 参数 | 说明 |
|------|------|
| `-o, --output FILE` | 输出文件路径（默认 stdout） |
| `--json` | 输出 JSON 格式而非 Markdown |
| `--page N` | 仅提取第 N 页（0-indexed） |
| `--no-separator` | 不添加页分隔符 |
| `--timeout SECONDS` | 请求超时秒数（默认 120） |
| `--orientation` | 启用文档方向分类 |
| `--unwarp` | 启用文档展平 |
| `--chart` | 启用图表识别 |
| `-q, --quiet` | 静默模式，不输出进度信息 |
| `--config FILE` | 指定配置文件路径 |

### configure 子命令参数

| 参数 | 说明 |
|------|------|
| `--server-url URL` | 设置服务器地址 |
| `--token TOKEN` | 设置访问令牌 |
| `-s, --scope SCOPE` | 配置保存范围：user（默认）、project、local |
| `--show` | 显示当前配置 |
| `--test` | 测试服务器连接 |
| `--locations` | 显示配置文件搜索路径 |

## 支持格式

PDF, PNG, JPG, JPEG, BMP, TIFF, WebP
