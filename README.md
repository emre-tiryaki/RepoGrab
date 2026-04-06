# RepoGrab

RepoGrab is a command-line utility designed to facilitate the selective download of specific files from both GitHub and GitLab repositories. It provides an interactive Terminal User Interface (TUI) for an intuitive user experience, enabling users to browse repository contents and download desired items with parallel processing capabilities.

## Features

*   **Interactive TUI**: Navigate and select files using a user-friendly terminal interface.
*   **Multi-Platform Support**: Seamlessly download files from GitHub and GitLab repositories.
*   **Selective Downloads**: Choose individual files or entire directories to download.
*   **Parallel Downloading**: Leverages Go's concurrency model to download multiple files simultaneously, enhancing performance.
*   **Access Token Management**: Configure and store access tokens for private repositories or to overcome API rate limits.
*   **Customizable Download Path**: Specify a preferred local directory for downloaded content.

## Inspiration

This project draws inspiration from [ghgrab](https://github.com/abhixdd/ghgrab). While the original `ghgrab` project is implemented in Rust and does not support parallel downloading, RepoGrab was developed as an alternative in Go, specifically to explore Go's concurrency features and implement parallel file downloads, offering a performance advantage for large selections.

## Getting Started

To use RepoGrab, you will need to have Go installed on your system (version 1.26.1 or newer, as specified in `go.mod`).

### Installation

1.  **Clone the repository**:
    ```bash
    git clone https://github.com/emre-tiryaki/repograb.git
    cd repograb
    ```
2.  **Build and Install**:
    ```bash
    go install ./cmd/repograb
    ```
    This command will compile the application and place the `repograb` executable in your `$GOPATH/bin` directory, making it accessible from your command line.

### Usage

RepoGrab operates through an interactive Terminal User Interface (TUI).

1.  **Launch the application**:
    ```bash
    repograb
    ```

2.  **Enter Repository URL**:
    *   The application will prompt you to enter a GitHub or GitLab repository URL.
    *   Example: `https://github.com/user/repo` or `https://gitlab.com/group/project`
    *   **Controls**:
        *   `[Enter]`: Confirm the URL and proceed.
        *   `[t]`: Access token settings.
        *   `[p]`: Configure the download folder.
        *   `[q]`: Quit the application.

3.  **Access Token Settings (Optional)**:
    *   If you need to access a private repository or encounter API rate limits, you can set an access token.
    *   **Controls**:
        *   `[Tab]`: Switch between GitHub and GitLab token providers.
        *   `[Enter]`: Save the entered token.
        *   `[Esc]`: Cancel and return to the previous screen.
        *   `[q]`: Quit the application.

4.  **Download Folder Settings (Optional)**:
    *   Specify the local directory where files will be downloaded.
    *   **Assumption**: The default download directory is typically the current working directory or a user's standard downloads folder.
    *   **Controls**:
        *   `[Enter]`: Save the specified path.
        *   `[Esc]`: Cancel and return to the previous screen.
        *   `[q]`: Quit the application.

5.  **Browse and Select Files**:
    *   After successfully processing the repository URL, RepoGrab will display a list of files and directories.
    *   **Controls**:
        *   `[Up/Down Arrow Keys]`: Navigate through the list.
        *   `[Space]`: Toggle selection of a file or directory. Selected items are marked with `[x]`.
        *   `[d]`: Initiate the download of all selected items.
        *   `[b]`: Go back to the repository URL input screen.
        *   `[q]`: Quit the application.

6.  **Downloading**:
    *   Once you initiate a download, the application will display a loading message.
    *   **Controls**:
        *   `[q]`: Quit the application (downloads may continue in the background depending on OS behavior, but the TUI will exit).

## Contributing

We welcome contributions and suggestions for RepoGrab. If you have a feature request, bug report, or would like to contribute code, please open an issue on the GitHub repository.
