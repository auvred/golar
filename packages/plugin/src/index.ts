import os from 'node:os'
import process from 'node:process'
import assert from 'node:assert'
import worker_threads from 'node:worker_threads'

const HEADER_SIZE = 5

const MSG_KIND = {
	CREATE_SERVICE_CODE: 0,
	CREATE_SERVICE_CODE_RESPONSE: 1,
}

const SERVICE_CODE_PROPERTIES = {
	SOURCE_MAP: 1 << 0,
}

const SCRIPT_KIND = {
	js: 0,
	jsx: 1,
	ts: 2,
	tsx: 3,
} as const satisfies Record<ScriptKind, number>

export type Mapping = {
	sourceOffset: number
	serviceOffset: number
	sourceLength: number
	serviceLength?: number | undefined
}

export type IgnoreDirectiveMapping = {
	serviceOffset: number
	serviceLength: number
}

export type ScriptKind = 'js' | 'jsx' | 'ts' | 'tsx'
export type ServiceCode = {
	serviceText: string
	scriptKind: ScriptKind
} & ({
	sourceMap: string
	mappings?: never
} | {
	sourceMap?: never
	mappings: Mapping[]
	ignoreMappings?: IgnoreDirectiveMapping[] | undefined
})

export type Promisable<T> = T | Promise<T>

export type CreatePluginOptions = {
	filename: string
	/**
		* @example ['.vue']
		*/
	extraExtensions?: string[] | undefined
	createServiceCode: (fileName: string, sourceText: string) => Promisable<ServiceCode>
}

export function createPlugin(opts: CreatePluginOptions) {
	if (worker_threads.isMainThread) {
		const workers = new Array(Math.max(Math.min(os.cpus().length / 2, 4), 1))
			.fill(null)
			.map(() => {
				const w = {
					busy: false,
					worker: new worker_threads.Worker(opts.filename)
						.on('message', () => {
							const task = taskQueue.shift()
							if (task == null) {
								w.busy = false
							} else {
								w.worker.postMessage(task)
							}
						})
				}
				return w
			})

		// TODO(perf): more perfomant FIFO?
		const taskQueue: Buffer[] = []

		let recvBuffer = Buffer.allocUnsafe(1024 * 1024)
		let recvBufferLen = 0
		function ensureRecvBuffer(n: number) {
		  if (recvBuffer.buffer.byteLength < n) {
		    const newBuffer = Buffer.allocUnsafe(n)
				recvBuffer.copy(newBuffer)
				recvBuffer = newBuffer
		  }
		}
		{
			const initialization = JSON.stringify({
				extraExtensions: opts.extraExtensions ?? [],
			})
			recvBuffer.writeUint32LE(Buffer.byteLength(initialization))
			process.stdout.write(recvBuffer.subarray(0, 4))
			process.stdout.write(initialization)
		}

		process.stdin.on('data', data => {
			assert.ok(data instanceof Buffer, 'Data is expected to be buffer')
			ensureRecvBuffer(recvBufferLen + data.byteLength)
			data.copy(recvBuffer, recvBufferLen)
			recvBufferLen += data.byteLength

			let msgBuffer = recvBuffer.subarray(0, recvBufferLen)
			let readOffset = 0
			while (true) {
				if (msgBuffer.byteLength < HEADER_SIZE) {
					break
				}
				const payloadLen = msgBuffer.readUInt32LE(1)
				if (msgBuffer.byteLength < payloadLen + HEADER_SIZE) {
					break
				}

				const worker = workers.find(({ busy }) => !busy)
				if (worker != null) {
					worker.busy = true
					worker.worker.postMessage(msgBuffer.subarray(0, payloadLen + HEADER_SIZE))
				} else {
					taskQueue.push(Buffer.copyBytesFrom(msgBuffer, 0, payloadLen + HEADER_SIZE))
				}
				readOffset += payloadLen + HEADER_SIZE
				msgBuffer = msgBuffer.subarray(payloadLen + HEADER_SIZE)
			}
			if (readOffset > 0) {
				recvBufferLen -= readOffset
				recvBuffer.copyWithin(0, readOffset)
			}
		});

	} else {
		const { parentPort } = worker_threads
		assert.ok(parentPort)
		let sendBuffer = Buffer.allocUnsafe(1024 * 1024)
		function prepareSendBuffer(msgKind: number, payloadLen: number) {
		  if (sendBuffer.buffer.byteLength < HEADER_SIZE + payloadLen) {
		    sendBuffer = Buffer.allocUnsafe(HEADER_SIZE + payloadLen)
		  }
			sendBuffer.writeUInt8(msgKind)
			sendBuffer.writeUInt32LE(payloadLen, 1)
			return sendBuffer.subarray(0, HEADER_SIZE + payloadLen)
		}
		parentPort.on('message', async function (data: Uint8Array)  {
			const buf = Buffer.from(data)
			const msgKind = buf.readUInt8(0)
			switch (msgKind) {
				case MSG_KIND.CREATE_SERVICE_CODE: {
					let offset = HEADER_SIZE
					const reqId = buf.readBigUInt64LE(offset)

					const fileNameLen = buf.readUInt32LE(offset += 8)
					const fileName = buf.subarray(offset += 4, offset + fileNameLen).toString('utf8')
					offset += fileNameLen

					const sourceTextLen = buf.readUInt32LE(offset)
					const sourceText = buf.subarray(offset += 4, offset + sourceTextLen).toString('utf8')
					offset += sourceTextLen
					const serviceCode = await opts.createServiceCode(fileName, sourceText)

					if ('sourceMap' in serviceCode) {
						const serviceTextLen = Buffer.byteLength(serviceCode.serviceText)
						const sourceMapLen = Buffer.byteLength(serviceCode.sourceMap)

						const sendBuffer = prepareSendBuffer(MSG_KIND.CREATE_SERVICE_CODE_RESPONSE, 8 + 1 + 1 + 4 + serviceTextLen + 4 + sourceMapLen)
						offset = HEADER_SIZE
						offset = sendBuffer.writeBigUInt64LE(reqId, offset)
						offset = sendBuffer.writeUInt8(SERVICE_CODE_PROPERTIES.SOURCE_MAP, offset)
						offset = sendBuffer.writeUInt8(SCRIPT_KIND[serviceCode.scriptKind], offset)
						offset = sendBuffer.writeUInt32LE(serviceTextLen, offset)
						offset += sendBuffer.write(serviceCode.serviceText, offset)
						offset = sendBuffer.writeUInt32LE(sourceMapLen, offset)
						offset = sendBuffer.write(serviceCode.sourceMap, offset)

						process.stdout.write(Buffer.copyBytesFrom(sendBuffer))
						parentPort.postMessage(null)
					} else {
						const serviceTextLen = Buffer.byteLength(serviceCode.serviceText)
						const mappingsLen = serviceCode.mappings.length * (4 * 4)
						const ignoreMappingsLen = (serviceCode.ignoreMappings?.length ?? 0) * 8

						const sendBuffer = prepareSendBuffer(MSG_KIND.CREATE_SERVICE_CODE_RESPONSE, 8 + 1 + 1 + 4 + serviceTextLen + 4 + mappingsLen + 4 + ignoreMappingsLen)

						offset = HEADER_SIZE
						offset = sendBuffer.writeBigUInt64LE(reqId, offset)
						offset = sendBuffer.writeUInt8(0, offset)
						offset = sendBuffer.writeUInt8(SCRIPT_KIND[serviceCode.scriptKind], offset)
						offset = sendBuffer.writeUInt32LE(serviceTextLen, offset)
						offset += sendBuffer.write(serviceCode.serviceText, offset)
						offset = sendBuffer.writeUInt32LE(serviceCode.mappings.length, offset)
						const writeUint32 = (os.endianness() === "LE" ? sendBuffer.writeUint32LE : sendBuffer.writeUint32BE).bind(sendBuffer)
						for (const m of serviceCode.mappings) {
							for (const i of [m.sourceOffset, m.serviceOffset, m.sourceLength, m.serviceLength ?? m.sourceLength]) {
								offset = writeUint32(i, offset)
							}
						}

						offset = sendBuffer.writeUInt32LE(serviceCode.ignoreMappings?.length ?? 0, offset)
						for (const m of serviceCode.ignoreMappings ?? []) {
							offset = writeUint32(m.serviceOffset, offset)
							offset = writeUint32(m.serviceLength, offset)
						}

						process.stdout.write(Buffer.copyBytesFrom(sendBuffer))
						parentPort.postMessage(null)
					}
				}
			}
		})
	}
}
