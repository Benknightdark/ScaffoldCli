# ScaffoldCli
- 透過cli將mssql的table轉成golang的struct
- 使用Multi Threading提升處理速度
# 安裝方式
```
go get -u github.com/Benknightdark/ScaffoldCli
```
# 使用方式
- 產生struct檔案指令
```
ScaffoldCli  \
-p "檔案儲存路徑" \
-m "模組名稱" \
-s "資料庫伺服器名稱" \
--po "Port Number (default: "1433")" \
-u "登入帳號" \
--pa "登入密碼" \
-d "資料庫名稱" 
```
- 查看指令用法
```
ScaffoldCli -h
```