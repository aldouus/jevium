# Files and media on physical iPhones

Provision an explicit local fixture before the goal:

```
jevium --udid DEVICE --fixture './sample.pdf=@com.example.documents:documents/sample.pdf' --goal 'Upload sample.pdf using Browse and On My iPhone'
```

The target app must be installed and expose its Documents container through iOS file sharing. The command overwrites the exact configured destination. Jevium validates all local fixtures before connecting, pushes each once, and pulls it back to verify its bytes. A failed push is never replayed. A failed verification stops the run and reports that the file may already have been written. Maximum fixture size is 20 MiB.

PDFs, images, and other file types can be provisioned this way. Browse/Files selection and existing Photo Library selection use current observed native buttons, cells, and images; the model never receives host file paths as selectors. Include the visible fixture filename in the goal. A successful transfer alone does not prove an upload completed; verify the site's resulting state.

Physical iOS does not expose an Appium API that imports a pushed image into the Photos database. An AFC media-folder write is not a Photos import, so this implementation deliberately rejects that destination. Select a pre-existing photo, or choose the provisioned image through Files. Camera capture, iCloud download management, Photos library import, arbitrary app sandbox access, and a synthetic browser file-input setter remain unsupported. Appium and iOS errors are returned directly rather than silently substituting another workflow.
