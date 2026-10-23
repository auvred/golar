import { make12, type Box12 } from './file12.ts'

export interface Box11<T> {
	value: T
	map<U>(fn: (value: T) => U): Box11<U>
}

export function make11<T>(value: T): Box11<T> {
	return { value, map: (fn) => make11(fn(value)) }
}

export class Store11<K extends string, V> {
	private readonly items = new Map<K, V>()
	set(key: K, value: V): this {
		this.items.set(key, value)
		return this
	}
	get(key: K): V | undefined {
		return this.items.get(key)
	}
	entries(): Array<[K, V]> {
		return [...this.items.entries()]
	}
}

export function use11(other: Box12<number>) {
	const store = new Store11<'a' | 'b', Box12<number>>()
	store.set('a', other).set('b', make12(other.value + 11))
	const mapped = make11(other.value)
		.map((n) => `${n}`)
		.map((s) => s.length)
	const total = store
		.entries()
		.reduce((sum, [, box]) => sum + box.value, mapped.value)
	const pairs = Object.entries({ x: total, y: mapped.value }).map(([k, v]) => ({
		k,
		v,
	}))
	return pairs.filter(({ v }) => v > 0).map(({ k }) => k.toUpperCase())
}
