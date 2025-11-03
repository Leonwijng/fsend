To do list for server

1 Client must be able to connect to the server (the executable, localhost; later hosted on an ICE server that streams packets between clients)

Server must return a list of available files

Current upload function still needs a receiver implementation

On every start of fsend, generate a UID

Store the UID locally and never delete it

UID acts as the identity for sending and receiving

Client checks on startup if the UID already exists (read-only)

Must support Windows and Linux

Eventually allow downloading files from the server