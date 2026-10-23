import { make13, type Box13 } from './file13.ts'

export interface Box12<T> {
	value: T
	map<U>(fn: (value: T) => U): Box12<U>
}

export function make12<T>(value: T): Box12<T> {
	return { value, map: (fn) => make12(fn(value)) }
}

export class Store12<K extends string, V> {
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

export function use12(other: Box13<number>) {
	const store = new Store12<'a' | 'b', Box13<number>>()
	store.set('a', other).set('b', make13(other.value + 12))
	const mapped = make12(other.value)
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
