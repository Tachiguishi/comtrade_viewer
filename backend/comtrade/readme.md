Parse COMTRADE files in Go.

## .CFG(Configuration File - 配置文件)

```
station_name,rec_dev_id,rev_year <CR/LF>
TT,##A,##D <CR/LF>
An,ch_id,ph,ccbm,uu,a,b,skew,min,max,primary,secondary,PS <CR/LF>
An,ch_id,ph,ccbm,uu,a,b,skew,min,max,primary,secondary,PS <CR/LF>
An,ch_id,ph,ccbm,uu,a,b,skew,min,max,primary,secondary,PS <CR/LF>
An,ch_id,ph,ccbm,uu,a,b,skew,min,max,primary,secondary,PS <CR/LF>
Dn,ch_id,ph,ccbm,y <CR/LF>
Dn,ch_id,ph,ccbm,y <CR/LF>
lf <CR/LF>
nrates <CR/LF>
samp,endsamp <CR/LF>
samp,endsamp <CR/LF>
dd/mm/yyyy,hh:mm:ss.ssssss <CR/LF>
dd/mm/yyyy,hh:mm:ss.ssssss <CR/LF>
ft <CR/LF>
timemult <CR/LF>
```

逐行详解：
- 第1行：电站、设备和标准版本
  - station_name: 变电站名称（例如: "Substation_A"）
  - rec_dev_id: 录波设备ID（例如: "Relay_B2"）
  - rev_year: 使用的 Comtrade 标准年份（例如: "1999" 或 "2013"）

- 第2行：通道计数 TT
  - 总通道数: Total Channels = NA + ND
  - NA: 模拟通道（Analog）数量
  - ND: 数字通道（Digital）数量

- 第3行 到 (NA+2) 行：模拟通道（Analog Channel）定义
  - An: 模拟通道序号（1..NA）
  - ch_id: 通道名称（例如: "Va", "Ia"）
  - ph: 相位标识（例如: "A", "B", "C", "N"）
  - ccbm: 被监视的电路元件
  - uu: 通道单位（例如: "V", "A", "kV", "kA"）
  - a: 乘法系数（斜率）
  - b: 加法偏移量（截距）
  - skew: 时间偏移（通常为 0）
  - min, max: 该通道在 .DAT 中原始数据的最小/最大值（用于校验）
  - primary: CT/PT 一次侧系数（例如 3000A）
  - secondary: CT/PT 二次侧系数（例如 5A）
  - PS: 'P' 表示一次侧值，'S' 表示二次侧值
  - 真实值计算: 真实物理值 = (a × DataValue) + b，其中 DataValue 为 .DAT 中的原始整数值

- 第 (NA+3) 行 到 (NA+ND+2) 行：数字通道（Digital Channel）定义
  - Dn: 数字通道序号（1..ND）
  - ch_id: 通道名称（例如: "Trip_Signal_A", "Breaker_Status"）
  - ph: 相位标识（可留空）
  - ccbm: 被监视的电路元件
  - y: 通道正常状态（0 或 1）

- 倒数第6行：电网频率
  - lf: 标称电网频率（例如: 50 或 60 Hz）

- 倒数第5行：采样率段数
  - nrates: 采样率变化次数（绝大多数场景为 1）

- 倒数第4行：采样率定义（若 nrates > 1，此处会有多行）
  - samp: 采样率（Hz，例如: 12800） 
  - endsamp: 该采样率持续到的最后样本序号

- 倒数第3行：开始时间
  - dd/mm/yyyy,hh:mm:ss.ssssss：记录中第一个采样点的绝对时间

- 倒数第2行：触发时间
  - dd/mm/yyyy,hh:mm:ss.ssssss：故障触发事件的绝对时间
  - 可通过与开始时间对比得出“故障前”数据长度

- 倒数第1行：数据文件格式
  - ft: 指明 .DAT 文件是 ASCII 还是 Binary

## .DAT (Data File - 数据文件)

.DAT 文件存储了所有通道在每个采样点的瞬时值。它没有“元数据”，纯粹是数据罗列。其格式由 .CFG 文件的最后一行 (ft) 决定。以下以 NA=2（模拟通道 2 个）、ND=1（数字通道 1 个）为例。

- 格式一：ASCII
  - 每一行代表一个采样点，逗号分隔：
    n, timestamp, A1, A2, ..., ANA, D1, D2, ..., DND
  - 字段说明：
    - n：样本序号（从 1 开始）
    - timestamp：相对时间戳（单位：μs），相对于 .CFG 中定义的“开始时间”
    - A1...ANA：NA 个模拟通道的原始整数值（需用 .CFG 中的 a、b 换算为物理值）
    - D1...DND：ND 个数字通道的值（0 或 1）
  - 示例（NA=2, ND=1）：
    ```
    1,0,12034,-8500,0
    2,78,12500,-9200,0
    3,156,13010,-9800,0
    4,234,13400,-10300,1   <-- 假设第 4 点数字通道 D1 从 0 变为 1
    5,312,13900,-11000,1
    ...
    ```
  - 优点：可读性强，便于调试
  - 缺点：文件体积大，读写速度慢

- 格式二：Binary（1999 版标准，16-bit）
  - 数据结构与 ASCII 相同，但按定长二进制紧凑存储（每个采样点的数据块连续、无分隔符）
  - 每个采样点的字段与类型：

| 数据 | 数据类型 | 字节数 |
| --- | --- | --- |
| 样本序号 (n) | 32位无符号整数 (unsigned long) | 4 字节 |
| 时间戳 (timestamp) | 32位无符号整数 (unsigned long) | 4 字节 |
| 模拟通道1 (A1) | 16位有符号整数 (short) | 2 字节 |
| 模拟通道2 (A2) | 16位有符号整数 (short) | 2 字节 |
| …（直到 ANA） |  |  |
| 数字通道 (D1...DND) | 16位无符号整数 (unsigned short) | 2 字节 |
  - 模拟通道 Ai：16 位有符号整数（int16，每个 2 字节，重复 NA 次）
  - 数字通道打包：16 位无符号整数（uint16，2 字节）× ceil(ND/16)
    - 若 ND ≤ 16：D1..DND 打包到 1 个 uint16
    - 若 16 < ND ≤ 32：使用 2 个 uint16，以此类推
    - 位位置信息：D1→bit0，D2→bit1，…，D16→bit15；D17 起放入下一个 uint16
  - 示例（NA=2, ND=1，十六进制）：
    ```
    [01 00 00 00]  n = 1
    [00 00 00 00]  timestamp = 0
    [F2 2E]        A1 = 12034
    [54 DE]        A2 = -8500
    [00 00]        D1 = 0

    [02 00 00 00]  n = 2
    [4E 00 00 00]  timestamp = 78
    ...            （依次类推）

    [04 00 00 00]  n = 4
    [EA 00 00 00]  timestamp = 234
    ...            （A1, A2 的值）
    [01 00]        D1 = 1（bit0 = 1）
    ```
  - 优点：文件体积小（通常比 ASCII 小 5–10 倍），读写速度快
  - 缺点：不便人工阅读，需结合 .CFG 解析

注：2013 版标准引入了 32-bit 整数与 32-bit 浮点等格式，但 1999 版 16-bit 整数二进制仍最常见。
