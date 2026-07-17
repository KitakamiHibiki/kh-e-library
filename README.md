# kitakami_hibiki e-library

瀵规爣 Google Play 鍥句功鐨勪釜浜虹綉缁滀簯绔功搴撱€備功绫嶅彲瀛樺偍浜庢湰鍦版垨鐧惧害缃戠洏銆?
## 姒傝堪

鑷墭绠＄殑 EPUB 鐢靛瓙涔︾鐞嗗钩鍙般€傞€氳繃 Web 鐣岄潰绠＄悊钘忎功銆佸湪绾块槄璇伙紝瀛樺偍鍚庣鍙嚜鐢遍€夋嫨鏈湴鏂囦欢绯荤粺鎴栫櫨搴︾綉鐩橈紝鏁版嵁涓绘潈褰掔敤鎴锋墍鏈夈€?
## 鎶€鏈爤

| 灞傜骇 | 閫夊瀷 |
|------|------|
| 鍚庣 | Go + Gin |
| 鍓嶇 | Vue 3 + TypeScript + Element Plus |
| 鏁版嵁搴?| SQLite (via GORM) |
| 瀛樺偍 | 鏈湴鏂囦欢绯荤粺 / 鐧惧害缃戠洏 Open API (鍙彃鎷? |
| 闃呰鍣?| 鍩轰簬 Web 鐨?EPUB.js |
| 閮ㄧ讲 | 鍗曟枃浠朵簩杩涘埗鎴?Docker |

## 鏋舵瀯

`
+----------------------------+
|        Web Browser          |
|  (涔︽灦 / 闃呰鍣?/ 绠＄悊闈㈡澘)  |
+-------------+--------------+
              | HTTP
+-------------v--------------+
|         Go Backend          |
|  +---------+ +------+ +--+ |
|  | 绠＄悊 API | | 闃呰  | |瀛榺 |
|  |         | | 鏈嶅姟  | |鎶絴 |
|  +---------+ +------+ +-+ |
+----------------------------+---+
              |                |
    +---------+--------+------+------+
    |                  |             |
+---v----+      +-----v----+  +-----v----+
| 鏈湴鏂囦欢绯荤粺 |   | 鐧惧害缃戠洏  |  | 鍏朵粬(棰勭暀) |
+----------+      +----------+  +----------+
`

## 鍔熻兘

| 鍔熻兘 | 璇存槑 |
|------|------|
| 涔︾睄绠＄悊 | 涓婁紶銆佸垹闄ゃ€佸垪琛ㄥ睍绀?|
| EPUB 闃呰 | 鍩轰簬娴忚鍣ㄧ殑闃呰鍣紝鏀寔涔︾銆佺瑪璁般€佽繘搴﹀悓姝?|
| 涔︽灦灞曠ず | 缃戞牸/鍒楄〃瑙嗗浘锛屾悳绱€佺瓫閫夈€佹帓搴?|
| 瀛樺偍鍚庣 | 鏈湴鏂囦欢绯荤粺 / 鐧惧害缃戠洏鍙垏鎹?|

## 蹇€熷紑濮?
### 鏈湴缂栬瘧

`ash
VERSION=v0.0.1
cd backend
go build -ldflags="-s -w -X main.Version=" -o e-library ./cmd/server
./e-library
`

### Docker 缂栬瘧閮ㄧ讲

`ash
docker build -f docker/Dockerfile.build \
  --build-arg E_LIBRARY_VERSION=v0.0.1 \
  -t e-library:build .
`

### Docker 涓嬭浇閮ㄧ讲

`ash
docker build -f docker/Dockerfile.download \
  --build-arg E_LIBRARY_VERSION=v0.0.1 \
  -t e-library:download .
`

### 鐜鍙橀噺

| 鍙橀噺 | 榛樿鍊?| 璇存槑 |
|------|--------|------|
| SERVER_HOST | 0.0.0.0 | 鐩戝惉鍦板潃 |
| SERVER_PORT | 14325 | 鐩戝惉绔彛 |
| DB_PATH | ./data/library.db | SQLite 鏁版嵁搴撹矾寰?|
| STORAGE_DRIVER | local | 瀛樺偍椹卞姩 (local / baidu) |
| STORAGE_LOCAL_BOOKS_DIR | ./data/books | 鏈湴涔︾睄瀛樺偍鐩綍 |

## 寮€鍙戣矾绾?
- [x] Phase 0锛氶」鐩剼鎵嬫灦锛屽悗绔熀纭€妗嗘灦锛屾暟鎹簱 schema
- [ ] Phase 1锛氭湰鍦板瓨鍌ㄥ悗绔紝EPUB 涓婁紶瑙ｆ瀽锛屼功鏋跺睍绀猴紝鍩虹闃呰鍣?- [ ] Phase 2锛氶槄璇昏繘搴﹀悓姝ワ紝涔︾銆佺瑪璁?- [ ] Phase 3锛氱櫨搴︾綉鐩樺瓨鍌ㄥ悗绔泦鎴?- [ ] Phase 4锛氭壒閲忓鍏ャ€丆alibre 闆嗘垚銆丱PDS
- [ ] Phase 5锛氱Щ鍔ㄧ閫傞厤銆丳WA

## License

MIT
