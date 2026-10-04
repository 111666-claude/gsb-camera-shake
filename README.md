# camera-shake

震屏合成：按种类覆写默认参数、多来源叠加并钳制、按剩余时间衰减、每秒强度预算与请求幂等，
只用 Go 标准库。

```
go run ./cmd/camera-shake --sample=override
go run ./cmd/camera-shake --sample=clamp
go run ./cmd/camera-shake --sample=decay
go run ./cmd/camera-shake --sample=quota
go run ./cmd/camera-shake --sample=idem
go run ./cmd/camera-shake --sample=scan
go test ./...
go vet ./...
```

## 口径（README 为准）

- **参数覆盖层**：某个种类有覆写参数就按覆写算，否则用默认；来源强度等于该种类振幅乘 `scale`，
  持续时间等于该种类的 `DurationMs`。`scale` 不大于 0 的请求返回 0。
- **叠加与钳制**：同一时刻的多个来源强度相加，合成立场强度不超过 `maxTotal`。
- **衰减**：每个来源按已过时间占 `DurationMs` 的比例线性衰减，时间到就移除。
- **每秒预算**：同一整秒内成功生效的强度总和不超过 `budgetPerSec`，放不下的部分必须截断，
  `Push` 返回实际生效的强度；额度用完返回 0。配额按整秒分桶，进入下一秒重新计算。
- **幂等**：同一个 `reqID` 只生效一次，重复调用返回 0。
- **代价**：`Active` 只遍历当前活跃的来源，不许扫全部历史来源，`Scanned` 不随历史请求数放大；
  10 万个来源、每秒 10 万次查询。
- **不变量**：合成强度不超过 `maxTotal`；同一串请求重复执行结果相同。

## 输出契约（不改格式）

```
total=..
second=..
scanned=..
```

## 目录

```
shake.go              覆盖层、叠加、衰减、预算与幂等
shake_test.go         单元测试
cmd/camera-shake/     命令行入口
```
