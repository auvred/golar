import os from 'node:os'
import process from 'node:process'
import assert from 'node:assert'
import worker_threads from 'node:worker_threads'

const HEADER_SIZE = 5
const PROTOCOL_VERSION = 1

const MSG_KIND = {
	CREATE_SERVICE_CODE: 0,
	CREATE_SERVICE_CODE_RESPONSE: 1,
}

const SERVICE_CODE_PROPERTIES = {
	ERROR: 1 << 0,
	DECLARATION_FILE: 1 << 1,
	IGNORE_NOT_MAPPED_DIAGNOSTICS: 1 << 2,
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

export type ExpectErrorDirectiveMapping = {
	sourceOffset: number
	serviceOffset: number
	sourceLength: number
	serviceLength: number
}

export type ScriptKind = 'js' | 'jsx' | 'ts' | 'tsx'

export type ServiceCodeError = {
	message: string
	start: number
	end: number
}

export type ServiceCode =
	| {
			mappings: Mapping[]
			errors?: never
			serviceText: string
			scriptKind: ScriptKind
			/** @default false */
			declarationFile?: boolean | undefined
			ignoreMappings?: IgnoreDirectiveMapping[] | undefined
			expectErrorMappings?: ExpectErrorDirectiveMapping[] | undefined
			ignoreNotMappedDiagnostics?: boolean | undefined
	  }
	| {
			mappings?: never
			errors: ServiceCodeError[]
	  }

export type Promisable<T> = T | Promise<T>

export type Extension = {
	/** Include the leading dot, e.g. '.vue'. */
	extension: string
	stripFromDeclarationFileName: boolean
	allowExtensionlessImports: boolean
}

export type CreatePluginOptions = {
	filename: string
	extensions?: Extension[] | undefined
	createServiceCode: (
		cwd: string,
		configFileName: string | null,
		fileName: string,
		sourceText: string,
	) => Promisable<ServiceCode>
}

export function createPlugin(opts: CreatePluginOptions) {
	if (worker_threads.isMainThread) {
		const workers = new Array(
			Math.floor(Math.max(Math.min(os.availableParallelism() / 2, 4), 1)),
		)
			.fill(null)
			.map(() => {
				const w = {
					busy: false,
					worker: new worker_threads.Worker(opts.filename).on('message', () => {
						const task = taskQueue.shift()
						if (task == null) {
							w.busy = false
						} else {
							w.worker.postMessage(task)
						}
					}),
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
				protocolVersion: PROTOCOL_VERSION,
				extensions: opts.extensions ?? [],
			})
			recvBuffer.writeUint32LE(Buffer.byteLength(initialization))
			process.stdout.write(recvBuffer.subarray(0, 4))
			process.stdout.write(initialization)
		}

		process.stdin.on('end', () => process.exit(0))
		process.stdin.on('data', (data) => {
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
					worker.worker.postMessage(
						msgBuffer.subarray(0, payloadLen + HEADER_SIZE),
					)
				} else {
					taskQueue.push(
						Buffer.copyBytesFrom(msgBuffer, 0, payloadLen + HEADER_SIZE),
					)
				}
				readOffset += payloadLen + HEADER_SIZE
				msgBuffer = msgBuffer.subarray(payloadLen + HEADER_SIZE)
			}
			if (readOffset > 0) {
				recvBufferLen -= readOffset
				recvBuffer.copyWithin(0, readOffset)
			}
		})
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
		parentPort.on('message', async function (data: Uint8Array) {
			const buf = Buffer.from(data)
			const msgKind = buf.readUInt8(0)
			switch (msgKind) {
				case MSG_KIND.CREATE_SERVICE_CODE: {
					let offset = HEADER_SIZE
					const reqId = buf.readBigUInt64LE(offset)

					const cwdLen = buf.readUInt32LE((offset += 8))
					const cwd = buf
						.subarray((offset += 4), offset + cwdLen)
						.toString('utf8')
					offset += cwdLen

					const configFileNameLen = buf.readUInt32LE(offset)
					const configFileName = buf
						.subarray((offset += 4), offset + configFileNameLen)
						.toString('utf8')
					offset += configFileNameLen

					const fileNameLen = buf.readUInt32LE(offset)
					const fileName = buf
						.subarray((offset += 4), offset + fileNameLen)
						.toString('utf8')
					offset += fileNameLen

					const sourceTextLen = buf.readUInt32LE(offset)
					const sourceText = buf
						.subarray((offset += 4), offset + sourceTextLen)
						.toString('utf8')
					offset += sourceTextLen
					const resp = await opts.createServiceCode(
						cwd,
						configFileName === '/dev/null/inferred' ? null : configFileName,
						fileName,
						sourceText,
					)

					let properties = 0

					if ('errors' in resp) {
						const errorLocationsUtf8 = mapIndicesToUtf8(
							sourceText,
							resp.errors[Symbol.iterator]().flatMap((e) => [e.start, e.end]),
						)
						const errorsLen = resp.errors.reduce(
							(sum, e) => sum + 4 + Buffer.byteLength(e.message) + 4 + 4,
							0,
						)
						const sendBuffer = prepareSendBuffer(
							MSG_KIND.CREATE_SERVICE_CODE_RESPONSE,
							8 + 1 + 4 + errorsLen,
						)
						offset = HEADER_SIZE
						offset = sendBuffer.writeBigUInt64LE(reqId, offset)
						offset = sendBuffer.writeUInt8(
							properties | SERVICE_CODE_PROPERTIES.ERROR,
							offset,
						)
						offset = sendBuffer.writeUInt32LE(resp.errors.length, offset)
						for (const err of resp.errors) {
							offset = sendBuffer.writeUInt32LE(
								Buffer.byteLength(err.message),
								offset,
							)
							offset += sendBuffer.write(err.message, offset)
							offset = sendBuffer.writeUInt32LE(
								errorLocationsUtf8(err.start)!,
								offset,
							)
							offset = sendBuffer.writeUInt32LE(
								errorLocationsUtf8(err.end)!,
								offset,
							)
						}

						process.stdout.write(Buffer.copyBytesFrom(sendBuffer))
						parentPort.postMessage(null)
						return
					}

					if (resp.declarationFile) {
						properties |= SERVICE_CODE_PROPERTIES.DECLARATION_FILE
					}
					if (resp.ignoreNotMappedDiagnostics) {
						properties |= SERVICE_CODE_PROPERTIES.IGNORE_NOT_MAPPED_DIAGNOSTICS
					}

					const serviceTextLen = Buffer.byteLength(resp.serviceText)
					const mappingsLen = resp.mappings.length * (4 * 4)
					const ignoreMappingsLen = (resp.ignoreMappings?.length ?? 0) * (4 * 2)
					const expectErrorMappingsLen =
						(resp.expectErrorMappings?.length ?? 0) * (4 * 4)

					const sendBuffer = prepareSendBuffer(
						MSG_KIND.CREATE_SERVICE_CODE_RESPONSE,
						8 +
							1 +
							1 +
							4 +
							serviceTextLen +
							4 +
							mappingsLen +
							4 +
							ignoreMappingsLen +
							4 +
							expectErrorMappingsLen,
					)
					const writeUint32 = (
						os.endianness() === 'LE'
							? sendBuffer.writeUint32LE
							: sendBuffer.writeUint32BE
					).bind(sendBuffer)

					offset = HEADER_SIZE
					offset = sendBuffer.writeBigUInt64LE(reqId, offset)
					offset = sendBuffer.writeUInt8(properties, offset)
					offset = sendBuffer.writeUInt8(SCRIPT_KIND[resp.scriptKind], offset)
					offset = sendBuffer.writeUInt32LE(serviceTextLen, offset)
					offset += sendBuffer.write(resp.serviceText, offset)

					const sourceIndicesUtf8 = mapIndicesToUtf8(
						sourceText,
						(function* () {
							for (const m of resp.mappings) {
								yield m.sourceOffset
								yield m.sourceOffset + m.sourceLength
							}
							for (const m of resp.expectErrorMappings ?? []) {
								yield m.sourceOffset
								yield m.sourceOffset + m.sourceLength
							}
						})(),
					)
					const serviceIndicesUtf8 = mapIndicesToUtf8(
						resp.serviceText,
						(function* () {
							for (const m of resp.mappings) {
								yield m.serviceOffset
								yield m.serviceOffset + (m.serviceLength ?? m.sourceLength)
							}
							for (const m of resp.ignoreMappings ?? []) {
								yield m.serviceOffset
								yield m.serviceOffset + m.serviceLength
							}
							for (const m of resp.expectErrorMappings ?? []) {
								yield m.serviceOffset
								yield m.serviceOffset + m.serviceLength
							}
						})(),
					)

					offset = sendBuffer.writeUInt32LE(resp.mappings.length, offset)
					for (const m of resp.mappings) {
						const sourceOffsetUtf8 = sourceIndicesUtf8(m.sourceOffset)!
						const serviceOffsetUtf8 = serviceIndicesUtf8(m.serviceOffset)!
						offset = writeUint32(sourceOffsetUtf8, offset)
						offset = writeUint32(serviceOffsetUtf8, offset)
						offset = writeUint32(
							sourceIndicesUtf8(m.sourceOffset + m.sourceLength)! -
								sourceOffsetUtf8,
							offset,
						)
						offset = writeUint32(
							serviceIndicesUtf8(
								m.serviceOffset + (m.serviceLength ?? m.sourceLength),
							)! - serviceOffsetUtf8,
							offset,
						)
					}

					offset = sendBuffer.writeUInt32LE(
						resp.ignoreMappings?.length ?? 0,
						offset,
					)
					for (const m of resp.ignoreMappings ?? []) {
						const serviceOffsetUtf8 = serviceIndicesUtf8(m.serviceOffset)!
						offset = writeUint32(serviceOffsetUtf8, offset)
						offset = writeUint32(
							serviceIndicesUtf8(m.serviceOffset + m.serviceLength)! -
								serviceOffsetUtf8,
							offset,
						)
					}

					offset = sendBuffer.writeUInt32LE(
						resp.expectErrorMappings?.length ?? 0,
						offset,
					)
					for (const m of resp.expectErrorMappings ?? []) {
						const sourceOffsetUtf8 = sourceIndicesUtf8(m.sourceOffset)!
						const serviceOffsetUtf8 = serviceIndicesUtf8(m.serviceOffset)!
						offset = writeUint32(sourceOffsetUtf8, offset)
						offset = writeUint32(serviceOffsetUtf8, offset)
						offset = writeUint32(
							sourceIndicesUtf8(m.sourceOffset + m.sourceLength)! -
								sourceOffsetUtf8,
							offset,
						)
						offset = writeUint32(
							serviceIndicesUtf8(m.serviceOffset + m.serviceLength)! -
								serviceOffsetUtf8,
							offset,
						)
					}

					process.stdout.write(Buffer.copyBytesFrom(sendBuffer))
					parentPort.postMessage(null)
				}
			}
		})
	}
}

function mapIndicesToUtf8(text: string, indices: Iterable<number>) {
	const mapped = new Map<number, number>()
	const sorted = Array.from(new Set<number>(indices)).sort((a, b) => a - b)
	let utf8Offset = 0
	for (const [i, idx] of sorted.entries()) {
		mapped.set(
			idx,
			(utf8Offset += Buffer.byteLength(text.slice(sorted[i - 1] ?? 0, idx))),
		)
	}
	return mapped.get.bind(mapped)
}
