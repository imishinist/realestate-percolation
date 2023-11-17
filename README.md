# realestate percolation

## setup

```
./bin/index.sh realestate seed/realestate.jsonl
./bin/put-index.sh realestate_query config/realestate_query.mapping.json
./bin/index.sh realestate seed/realestate.jsonl
./bin/index.sh realestate_query seed/realestate_query.jsonl
```

## search query

指定した緯度経度にマッチするクエリを検索

指定した緯度経度

`GET realestate_query/_search`

```json
{
  "query": {
    "percolate": {
      "field": "query",
      "document": {
        "location": {
          "lat": 33.556691857517485,
          "lon": 130.42594518963125
        }
      }
    }
  }
}
```

クエリの名前に "大橋" と含まれるクエリを検索

`GET realestate_query/_search`

```json
{
  "query": {
    "match": {
      "message": "大橋"
    }
  }
}
```
