# WIP/POC - Simple Selective Forwarding Unit

A simple selective forwarding unit (SFU) for WebRTC written in Go.
Designed for local network use for distribute computer vision systems.
Does not include any authentication or authorization.

Signalling server to handle client connections
All clients only make one connection to the server
source 1 -->
source 2 --> server --> client 1
--> client 2
Reducing complexity and the number of connections each peer has to make by removing peer to peer connections.

## Features

- Forwarding between multiple peers.

## Viewers

- Readonly
- As all clients handle the offer. A viewer must add transceivers for video and audio to receive the tracks from the sources. Without this the viewer connection will fail. See viewer.html for an example.

## Dev Notes

- Does not currently allow specific tracks to be forwarded from sources to viewers.
- Does not handle disconnects gracefully.
- Does not handle renegotiation in source.html
- Renegotiation for source reload working, but not for viewer reload.

## Limitations

- No authentication or authorization.
- No transcoding.

## Notes

- You must renegotiate when you adjust the tracks of a peer.
