# goRed

goRed is a remote-dictionary service that speaks RESP(Redis Serialization Protocol).
goRed doesn't have its own client as of now. Since it speaks RESP, `redis-cli`
can be used as a client.

## Performance Benchmarking
The `benchmark.sh` script launches 50 parallel clients, each of which run the `get` and `set` commands 2000 times in parallel.

| Run No.| goRed | redis |
| --- | ------- | --------- |
| 1   | 1.329 sec | 1.226 sec |
| 2   | 1.286 sec | 1.219 sec |
| 3   | 1.264 sec | 1.214 sec |
| 4   | 1.300 sec | 1.236 sec |
| 5   | 1.270 sec | 1.241 sec |

Average execution time of `goRed`: 1.289 sec
<br>
Average execution time of `redis`: 1.227 sec

## Supported Commands

### PING
```
PING [string]
```
PING responds back with "PONG". It echoes back the optional string if passed.<br>
Example:
```
% redis-cli PING
PONG
% redis-cli PING "echo"
echo
```
TC: O(1)

### ECHO
```
ECHO <string>
```
Echoes back the string passed.<br>
Example:
```
% redis-cli ECHO "hello, world"
"hello, world" 
```
TC: O(1)

### GET
```
GET <key>
```
GET responds with the string value stored against "\<key\>".
It responds back with a "nil" value if the key-value pair hasn't been set.<br>
Note: GET responds back with an error if value stored isn't of type string.<br>
Example:
```
% redis-cli GET notset  
(nil)
% redis-cli SET name goRed
OK
% redis-cli GET name      
"goRed"
% redis-cli LPUSH list 1
(integer) 1
% redis-cli GET list    
(error) value is not of string type
```
TC: O(1)

### SET
```
SET <key> <value> [EX seconds | PX milliseconds | EXAT unix timestamp in seconds | PXAT unix timestamp in milliseconds]
```
SET is used to store a key-value pair in the remote dictionary, with an optional expiry time.
SET responds back with "OK" if the key has been set successfully.
SET can only be used to set string values.<br>
Options:<br>
1. `EX` - Expiry time in seconds
2. `PX` - Expiry time in milliseconds
3. `EXAT` - Expiry as a unix timestamp in seconds
4. `PXAT` - Expiry as a unix timestamp in milliseconds.
<br>
Example:
```
% redis-cli SET os linux
OK
% redis-cli GET os      
"linux"
% redis-cli SET os linux PX 100
OK
% redis-cli GET os             
(nil)
```
TC: O(1)

### EXISTS
```
EXISTS key [key...]
```
The EXISTS command is used to check the existence of a key(s).
EXISTS responds back with an integer which indicates the number of keys that exist out of the list of keys passed.
<br>
Example:
```
% redis-cli SET key1 value1
OK
% redis-cli EXISTS key1 key2
(integer) 1
```
TC: O(N), where "N" is the number of keys

### DEL
```
DEL key [key...]
```
DEL is used to delete a key(s). Non existent keys are ignored.
<br>
DEL responds back with the number of keys that were deleted.
<br>
Example:
```
% redis-cli SET key1 value1
OK
% redis-cli DEL key1 key2  
(integer) 1 
```
TC: O(N), where "N" is the number of keys.

### INCR
```
INCR key
```
INCR increments the value stored at key by 1.
If the value stored at key can't be converted to an integer type, an error is returned.<br>
If the key doesn't already exist, its added and incremented by one.
<br>
INCR responds back with the value post increment.
<br>
Example:
```
% redis-cli SET counter 0
OK
% redis-cli INCR counter 
(integer) 1
% redis-cli INCR counter
(integer) 2
% redis-cli INCR unsetcounter
(integer) 1
% redis-cli LPUSH list 1
(integer) 1
% redis-cli INCR list
(error) value is not of numeric type
```
TC: O(1)

### DECR
```
DECR key
```
DECR decrements the value stored at key by 1.
If the value stored at key can't be converted to an integer, an error is returned.<br>
If the key doesn't already exist, its added and decremented by one.
<br>
DECR responds back with the value post decrement.
<br>
Example:
```
% redis-cli SET counter 1
OK
% redis-cli DECR counter
(integer) 0
% redis-cli DECR unsetcounter
(integer) -1
% redis-cli LPUSH list 1
(integer) 1
% redis-cli DECR list
(error) value is not of numeric type
```
TC: O(1)

### LPUSH
```
LPUSH key element [element...]
```
The LPUSH command inserts all the values specified at the
head of a list stored at key. If key does not exist, it is created as empty list before performing the push operations. When key holds a value that is not a list, an error is returned.
<br>
LPUSH responds back with the number of elements inserted.
Exammple:
```
% redis-cli LPUSH list1 1 2 3 4 5
(integer) 5
% redis-cli SET counter 0
OK
% redis-cli LPUSH counter 1      
(error) value not of list type
```
TC: O(1) for each element

### RPUSH
```
RPUSH key element [element...]
```
The RPUSH command inserts all the values specified at the
tail of a list stored at key. If key does not exist, it is created as empty list before performing the push operations. When key holds a value that is not a list, an error is returned.
<br>
RPUSH responds back with the number of elements inserted.
<br>
Example:
```
% redis-cli RPUSH list 1 2 3 4 5
(integer) 5
% redis-cli SET counter 0      
OK
% redis-cli RPUSH counter 1
(error) value not of list type
```
TC: O(1) for each element

### LRANGE
```
LRANGE key start stop
```
LRANGE returns the specified elements of the list stored at key. The offsets start and stop are zero-based indexes, with 0 being the first element of the list (the head of the list), 1 being the next element and so on. "stop" can be greater than the size of the list. In case it is, elements from start to the end of the list are returned.
<br>
LRANGE responds with an error if the value stored at key isn't a list.
<br>
Example:
```
% redis-cli RPUSH list 1 2 3 4 5
(integer) 5
% redis-cli LRANGE list 0 10
1) "1"
2) "2"
3) "3"
4) "4"
5) "5"
% redis-cli SET counter 0   
OK
% redis-cli LRANGE counter 0 1
(error) value not of list type
```
TC: O(S + N), where "S" is the offset from the head of the list and "N" is the number of elements in the range.