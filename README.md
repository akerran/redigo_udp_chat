Redis backed chat room (Go language).

Tested functionality:
Chat room based on UDP protocol.
Clients send messages via UDP to Server. Then it is broadcast to all other clients
Server pushes message to Redis for temporary history. History is limited to 20 messages.
When new client connects to chat server it receives last 20 messages (in correct order)
Client may delete any message he/she has previously written (but not messages from others).
When client deletes message it is removed from Redis, new clients will see history without it.

Still needs to be done:
When client deletes message it is removed in chat screen for all clients.
When all clients disconnect DB is flushed
Unit tests.


Some implementation details:

Redis stores history using list 'messages' (it guarantees that all messages will be in correct order).
Every entry in 'messages' list has the following format:
<id> <username>: <message>

where:
<id> - some random 6 digit id of message
<username> - name that current user has chosen
<message> - text of message

Together with 'messages' list Redis stores hash for every user in the following format:
user:<username> <id> <entry from 'messages' list>

When user wants to remove his message, server will try to find message by message id in hash for current user. Then if message is found, it will be removed from 'messages' list.


