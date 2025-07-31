# OS AB Update Golang Version


## Getting Started

To get started with this project, clone the repository and navigate into the project directory:

```bash
git clone <repository-url>
cd <repo-name>
```

## Prerequisites

Make sure you have Go installed on your machine. You can download it from the official Go website.

## Running the Application

To run the application, use the following command:

```bash
go run cmd/main.go
```

The `main.go` file contains the entry point of the application. It initializes the application and starts the main process. Below are the possible Cobra commands defined in `main.go`:

### Commands

The application provides several commands to manage the OS update tool:

- write [update-image-path] [checksum]: Write rootfs partition with the specified update image path and checksum.    
- apply: Apply the updated image as the next boot.    
- commit: Commit the updated image as the default boot.    
- rollback: Rollback to the previous boot.    
- display: Display the current active partition.    

Make sure to review and understand the functionality implemented in `main.go` to effectively run and modify the application.   

Write rootfs partition:
```
go run cmd/main.go write /tmp/abc.raw.tar xx4454-dddf-dfdfd
```

Apply the updated image as the next boot.
```
go run cmd/main.go apply
```

Commit updated image as default boot:
```
go run cmd/main.go commit
```

Display current active partition:
```
go run cmd/main.go display
```


## Contributing

Contributions are welcome! Please open an issue or submit a pull request for any improvements or bug fixes.

## License

This project is licensed under the MIT License. See the LICENSE file for more details.
