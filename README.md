# CTP L3 订单簿估算器
<div align="center">
  <img src="demo.gif" alt="Demo GIF">
</div>

一个基于 Go 实现的实时 L3 订单簿可视化工具，用于国内期货期权合约，它尝试从 L2 市场深度数据重建单个订单队列。提供交互式可视化功能。

基于 @jose-donato 的实现：[binancef_l3_estimate_go](https://github.com/jose-donato/binancef_l3_estimate_go)

## ✨ 特性

### 🎯 **核心 L3 重建**
- 从 L2 数据实时重建单个订单队列
- 基于 FIFO 的队列管理与智能订单匹配
- 精确的小数运算，防止浮点错误
- 支持上期 CTP 提供的合约行情数据


### 🎨 **高级可视化**
- **基于年龄的着色**：颜色越深 = 订单越旧（队列靠前）
- **基于聚类的着色**：每个订单大小聚类使用不同颜色
- **特殊高亮**：最大和第二大订单使用金色高亮


### 📊 **交互式前端**
- 实时 D3.js 堆叠条形图
- 综合订单簿表格视图
- 队列可视化与单个订单条
- 交易对切换与实时更新
- 响应式设计，针对交易工作流优化


## 🚀 快速开始

### 选项 1: 使用运行脚本 (推荐)
```bash
./run.sh ag2510
```

### Option 2: Direct Go command
```bash
go run *.go au2510
```

Then open [http://localhost:8080](http://localhost:8080) in your browser.


## 📡 WebSocket API

The application exposes a WebSocket API for programmatic control:

```javascript
// Toggle clustering

// Switch symbol
ws.send(JSON.stringify({
    type: "switch_symbol", 
    symbol: "fu2510"
}));

```

## 🏗️ Architecture

```
┌─────────────────┐    ┌────────────────────┐    ┌─────────────────┐
│   Binance API   │────│  L3 Reconstruction │────│  Visualization  │
│  (L2 WebSocket) │    │     Algorithm      │    │   (D3.js + Go)  │
└─────────────────┘    └────────────────────┘    └─────────────────┘
                                │
                        ┌──────────────────┐
                        │   K-Means        │
                        │   Clustering     │
                        └──────────────────┘
```

## 🔬 L3 重建算法细节
该算法采用多种策略以准确还原订单队列：

1. **新增订单**：新订单插入队列尾部（FIFO）
2. **移除订单**：
   - 优先尝试精确匹配（如取消）
   - 数量变化较大时 → 移除最大订单
   - 数量变化较小时 → 从队头依次移除
3. **队列维护**：定期优化队列并更新订单年龄

指标追踪：全面的队列分析与统计

## 📦 Dependencies

- **Backend**: Go 1.23+, gorilla/websocket, shopspring/decimal
- **Frontend**: D3.js v7, vanilla JavaScript
- **Data Source**: Binance Futures WebSocket API

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.