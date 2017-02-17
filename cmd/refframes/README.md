# REFFRAME
## Purpose
Frame transformation is not trivial. NASA NAIF's SPICE does this incredibly well with high precision. Hence, SMD takes advantage of this in order to simplify the transformation between frames.

## Requirements
For simplicity, this batch of the tools relies on [SpiceyPy](https://github.com/AndrewAnnex/SpiceyPy). All the other requirements are in the `reqs.txt` file.

## Installation
In a new [virtual environment](docs.python-guide.org/en/latest/dev/virtualenvs/):
- `pip install -r reqs.txt`

## Tools
### chgframe
Allows to change between SPICE frames.
#### Usage
##### Input parameters
- `-s` (or `--state`): array of floats representing the radial and velocity vectors as [R, V]
- `-f` (or `--from`): SPICE frame name the provided `state` is currently defined in
- `-t` (or `--to`): SPICE frame name to transform the `state` to
- `-e` (or `--epoch`): date time of transformation

##### Output text
- An error occurred, the output starts with `Traceback`, followed by an error message. *Note: oddly, I can't seem to catch any SpiceyPy exception, hence the kind of ugly error.*
- Otherwise, the output will be an array of floats representing the state such as `[147374695.45440251, 22403597.978579231, 9716224.5381853618, -1.0054225372713503, 2.9031571735929518, 11.545995892127772]`
