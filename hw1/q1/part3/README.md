# part2

Note that we consider a broadcast to $N$ clients as $N$ separate send events. Further, when getting a total ordering of messages, if the timestamps are the same between two messages, a lower source ID is considered to be earlier than a higher source ID.
