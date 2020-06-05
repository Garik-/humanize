# Humanize MIDI Velocity
## Idea
Build a database of MIDI files created by people. Scan and save a set of velocity for each note.
After this set can be used to automatically arrange the velocity in your midi file.

## Usage
Build a list of midi files:
```
find . -type f -name "*.mid" > list.txt
```
Create a database or use which is in the repository
```
scan -l list.txt -o drums.json
```
Humanize your midi file
```
humanize -d drums.json -i in.mid -o out.mid -min 25 -max 110
```
By changing the values ​​of min and max you can get a quiet, loud or balanced track