# x-agent 项目开发规范

## Go 代码规范

### Import 语句格式

所有 Go 文件的 import 语句必须按照以下规范组织,不同类型的包之间使用空行分隔:

```go
import (
    // 标准库包
    "context"
    "fmt"
    "time"
    
    // 外部依赖包 (第三方库)
    "github.com/gin-gonic/gin"
    "github.com/docker/docker/client"
    
    // 内部包 (本项目的包)
    "github.com/go-wyvern/x-agent/internal/api"
    "github.com/go-wyvern/x-agent/pkg/container"
)
```

**分组规则:**
1. **标准库包**: Go 标准库提供的包 (如 `fmt`, `context`, `time` 等)
2. **外部依赖包**: 第三方库和外部依赖 (如 `github.com/gin-gonic/gin`)
3. **内部包**: 本项目的包 (以 `github.com/go-wyvern/x-agent/` 开头的包)

**注意事项:**
- 每组内的包按字母顺序排列
- 三组之间必须用空行分隔
- 如果只有一组或两组,仍然要保持分组结构
