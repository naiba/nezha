let receivedLength = 0;
let expectedLength = 0;
let root;
let draftHandle;
let accessHandle;

const Operation = Object.freeze({
    WriteHeader: 1,
    WriteChunks: 2,
    DeleteFiles: 3
});

onmessage = async function (event) {
    try {
        const { operation, arrayBuffer, fileName } = event.data;

        switch (operation) {
            case Operation.WriteHeader: {
                const dataView = new DataView(arrayBuffer);
                expectedLength = Number(dataView.getBigUint64(4, false));
                receivedLength = 0;

                // Create a new temporary file
                root = await navigator.storage.getDirectory();
                draftHandle = await root.getFileHandle(fileName, { create: true });
                accessHandle = await draftHandle.createSyncAccessHandle();

                // Inform that file handle is created
                const dataChunk = arrayBuffer.slice(12);
                receivedLength += dataChunk.byteLength;
                accessHandle.write(dataChunk, { at: 0 });
                const progress = 'got handle';
                postMessage({ type: 'progress', progress: progress });
                break;
            }
            case Operation.WriteChunks: {
                if (!accessHandle) {
                    throw new Error('accessHandle is undefined');
                }

                const dataChunk = arrayBuffer;
                accessHandle.write(dataChunk, { at: receivedLength });
                receivedLength += dataChunk.byteLength;

                if (receivedLength === expectedLength) {
                    accessHandle.flush();
                    accessHandle.close();

                    const fileBlob = await draftHandle.getFile();
                    const blob = new Blob([fileBlob], { type: 'application/octet-stream' });

                    postMessage({ type: 'result', blob: blob, fileName: fileName });
                }
                break;
            }
            case Operation.DeleteFiles: {
                for await (const [name, handle] of root.entries()) {
                    if (handle.kind === 'file') {
                        await root.removeEntry(name);
                    } else if (handle.kind === 'directory') {
                        await root.removeEntry(name, { recursive: true });
                    }
                }
                break;
            }
        }
    } catch (error) {
        postMessage({ error: error.message });
    }
};
