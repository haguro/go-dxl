# Go-DXL

*Note: This library is still in early development. Features and API are subject to change. Use at your own risk.*

go-dxl is a Go library for interfacing with the ROBOTIS Dynamixel® actuators. It aims to include a set of packages to communicate with Dynamixel devices at different levels of abstraction. It currently contains the following packages:

1. protocol (In progress) - low level communication with Dynamixel actuators  using the Dynamixel Protocol 1.0 and 2.0.

## Features

- Zero external dependencies: All packages in this library will only ever depend on other packages within this library and/or on the Go standard library.
- Lightweight: All packages should aim to have minimal memory footprint and computational overhead [Planned]
- TinyGo support: Whenever possible, the packages shall be designed to support compilation to TinyGo (only for standard targets - microcontroller targets are under consideration and will be decided upon as more packages are added) [Planned]
- Simple API: The API shall aim to be simple and easy to use while exposing all functionality. [Planned]
- Abstraction layer support for all Dynamixel servo families: The protocol package will support low level communication for all Dynamixel servo families (AX, MX, XM, XH, PRO/PRO-M) via implementation of the Dynamixel protocol versions 1 [Planned] and 2 [Done]. Higher level packages wil abstract away differences between servo families. [Planned]
Servo simulator support: Allow simulation of Dynamixel servos and their response to commands without requiring physical hardware. [Planned]

## Contributing

Contributions are always welcome!

If you've found a bug, please  open an issue with details to reproduce it. If you'd like to contribute the fix, please submit a pull request with the changes.

New features as well as enhancement ideas are also welcome, but please open an issue first to discuss the proposed changes before implementing them. This will help ensure the changes are in line with the goals and architecture of the project.

## Disclaimer

This is an independent project and is not affiliated with or endorsed by ROBOTIS Co., Ltd. 'ROBOTIS' and its trademarks, including 'Dynamixel' and 'Dynamixel PRO', are the property of ROBOTIS Co., Ltd. The purpose of this project is to provide a client library to facilitate communication with Dynamixel actuators from Go programs. Any use of ROBOTIS trademarks in this project is for identification purposes only and does not indicate endorsement or affiliation with ROBOTIS.

## License

This project is licensed under the [MIT License](LICENSE).

## Warranty

This code library is provided "as is" and without any warranties whatsoever. Use at your own risk. More details in the [LICENSE](LICENSE) file.
