# ha-trajectory

一个用于接收并存储 Home Assistant 位置上报的 Go 后端服务，支持轨迹查询（指定日期 / 最近 N 天 / 全部）。

## 功能概览
- POST /api/track 接收 HA 位置上报
- GET /api/path 查询轨迹线（LineString）
- PostGIS 存储 geometry(Point, 4326)
- 支持 .env 读取 DB_DSN
- Docker 构建与部署

## 运行环境
- Go 1.25+
- PostgreSQL + PostGIS

## 配置
在项目根目录创建 .env：

```
DB_DSN=host=127.0.0.1 user=postgres password=postgres dbname=ha_trajectory port=5432 sslmode=disable
```

## 本地运行

```
go run ./cmd/server
```

若需要自定义端口：

PowerShell：
```
$env:PORT = "8090"
go run ./cmd/server
```

CMD：
```
set PORT=8090
go run ./cmd/server
```

## API

### 1) 上报轨迹点
POST /api/track

请求 JSON：
```
{
  "device_id": "device_tracker.my_phone",
  "latitude": 31.45,
  "longitude": 120.12,
  "timestamp": "2026-01-28T16:00:00Z"
}
```

示例（CMD）：
```
curl -X POST http://localhost:8080/api/track -H "Content-Type: application/json" -d "{\"device_id\":\"device_tracker.my_phone\",\"latitude\":31.45,\"longitude\":120.12,\"timestamp\":\"2026-01-28T16:00:00Z\"}"
```

### 2) 查询轨迹线
GET /api/path

参数：
- device_id 必填
- date=YYYY-MM-DD 指定某天
- days=N 最近 N 天
- all=1 或 all=true 全部
- 不传 date/days/all 时默认当天（UTC）

示例：
```
# 默认当天
curl "http://localhost:8080/api/path?device_id=device_tracker.my_phone"

# 指定日期
curl "http://localhost:8080/api/path?device_id=device_tracker.my_phone&date=2026-01-28"

# 最近 7 天
curl "http://localhost:8080/api/path?device_id=device_tracker.my_phone&days=7"

# 全部
curl "http://localhost:8080/api/path?device_id=device_tracker.my_phone&all=1"
```

返回示例：
```
{"device_id":"device_tracker.my_phone","date":"2026-01-28","geojson":"{\"type\":\"LineString\",\"coordinates\":[[120.12,31.45],[120.1205,31.4505]]}"}
```

## Docker

### 方案 A：直接打包本机二进制（scratch）
先编译 Linux 静态二进制：

PowerShell：
```
$env:GOOS="linux"
$env:GOARCH="amd64"
$env:CGO_ENABLED="0"
go build -ldflags="-s -w" -o ha-trajectory ./cmd/server
```

Dockerfile（示例）：
```
FROM scratch
COPY ha-trajectory /ha-trajectory
EXPOSE 8080
ENTRYPOINT ["/ha-trajectory"]
```

构建/运行：
```
docker build -t ha-trajectory:bin .
docker run --rm -p 8080:8080 --env-file .env ha-trajectory:bin
```


## 可视化
接口返回的是 GeoJSON LineString。若需要点 + 时间，可在前端将点列表与时间绑定后展示。

## 目录结构
```
cmd/server
internal/config
internal/handlers
internal/models
internal/repository
```
