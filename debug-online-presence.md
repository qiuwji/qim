# Debug Session: online-presence

Status: [OPEN]

## Symptom

- Typing display is correct.
- Online/offline display is still incorrect in the deployed QIM app.

## Hypotheses

1. Backend presence events are not pushed to friends when a user connects/disconnects.
2. `presence/online_friends` returns an incorrect map or omits online friends.
3. Frontend receives the correct presence data but overwrites it with default `false` during base refresh.
4. The deployed frontend/backend is not running the latest committed code.
5. Friend relationship data on the server differs from the accounts used for reproduction.

## Evidence Log

- Production logs show presence events are emitted.
- `uid=1` online/offline logs report `friend_count=1`.
- `uid=2` online/offline logs repeatedly report `friend_count=0`.
- Production SQLite data confirms `friend_requests` has accepted request `2 -> 1`, but `friends` has no `1 -> 2` or `2 -> 1` rows.

## Conclusion

- The presence pipeline is running, but it depends on `friends`.
- Current production data has an accepted friend request without corresponding bidirectional `friends` rows.
- Because `uid=2` has `friend_count=0`, its online/offline events are not pushed to `uid=1`.

## Fix

- Make accepting a friend request idempotently create bidirectional friend edges.
- Add startup repair for historical accepted friend requests missing friend edges.

## Verification

- `go test ./...` under `backend` passed after the fix.
