# Files and media on physical iPhones

Provision an explicit local fixture before the goal:

```
jevium --udid DEVICE --fixture './sample.pdf=@com.example.documents:documents/sample.pdf' --goal 'Upload sample.pdf using Browse and On My iPhone'
```

The target app must be installed and expose its Documents container through iOS file sharing. The command overwrites the exact configured destination. Jevium validates all local fixtures before connecting, pushes each once, and pulls it back to verify its bytes. A failed push is never replayed. A failed verification stops the run and reports that the file may already have been written. Maximum fixture size is 20 MiB.

PDFs, images, and other file types can be provisioned this way. Browse/Files selection and existing Photo Library selection use current observed native buttons, cells, and images; the model never receives host file paths as selectors. Include the visible fixture filename in the goal. A successful transfer alone does not prove an upload completed; verify the site's resulting state.

Physical iOS does not expose an Appium API that imports a pushed image into the Photos database. An AFC media-folder write is not a Photos import, so this implementation deliberately rejects that destination. Select a pre-existing photo, or choose the provisioned image through Files. Camera fixture injection, iCloud download management, Photos library import, arbitrary app sandbox access, and a synthetic browser file-input setter remain unsupported. Existing native camera controls remain ordinary observed click targets; Jevium does not replace the camera sensor with a fixture. Appium and iOS errors are returned directly rather than silently substituting another workflow.

## Retrieve downloads and artifacts

Repeat `--retrieve '@com.example.documents:documents/report.pdf=./report.pdf'` to pull configured files after the agent or inspector finishes, including a blocked goal or run error. The session remains open during retrieval. A failure returns a nonzero command result alongside any run error; previously saved artifacts remain available. Retrieval does not run if setup or initial observation fails.

Remote paths must name files in a shared Documents container. Local parent directories must already exist, and local destinations must not exist. Validation happens before connecting and again before retrieval. Transfers are limited to 20 MiB, decoded from validated base64, written to a temporary file in the destination directory, synced, and published without replacing an existing destination. A concurrent creation of the destination fails rather than overwriting it. No device write is involved and failed pulls are not retried.

This retrieves files already saved by the app into an accessible Documents container; it does not bypass Safari/iCloud sandbox restrictions or cause a download by itself. The natural-language goal must perform the app's visible download/save interaction first.
