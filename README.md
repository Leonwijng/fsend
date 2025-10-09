<p align="center">
  <img src="./.github/mascott.png" alt="mascott" />
</p>

# fsend

An application to send files to each other

## Features

- Send Files to server where recipient can download it

- Stream files from one client to another via intermediate server on udp

## Tranport protocol

The transport layer will be built on top of UDP which was chosen since clients dont need long lived connections.
The protocol will have metadata headers and the data payload. Each packet will contain a max of 1024 bytes of file data excluding metadata. Since udp is a connectionless protocol and does not maintain order and retries the protocol will also need to implements this.

first we send a udp packet that contains the sender and receiver uuid. we put this in a server side store and return a sender and receiver id.
on each packet the server will check if its a data packet or a init packet. the init packet will include the file name

Each packet will be constructed as follows:

```go
    // init packet
    +----------------------+
    | udp header (8 bytes) |
    +----------------------+
    +-------------------------------------+
    | packet type   uuid  (1 bytes)       |
    | sender_id     uuid  (16 bytes)      |
    | receiver_id   uuid  (16 bytes)      |
    | filename            (max 256 bytes) |
    | filesize            (8 bytes)       |
    +-------------------------------------+
    305 bytes total

    // data packet
    +----------------------+
    | udp header (8 bytes) |
    +----------------------+
    +--------------------------------+
    | packet type  uuid   (1 byte)   |
    | sender_id    uuid   (1 byte)   |
    | receiver_id  uuid   (1 byte)   |
    | fileid       uuid   (16 bytes) |
    | chunknumber  uint64 (8 bytes)  |
    +--------------------------------+
    +-----------------------------+
    | Data layer (max 1024 bytes) |
    +-----------------------------+

    1045 bytes total
```
