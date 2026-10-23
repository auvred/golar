import { make6, type Box6 } from './file6.ts'

export interface Box5<T> {
	value: T
	map<U>(fn: (value: T) => U): Box5<U>
}

export function make5<T>(value: T): Box5<T> {
	return { value, map: (fn) => make5(fn(value)) }
}

export class Store5<K extends string, V> {
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

export function use5(other: Box6<number>) {
	const store = new Store5<'a' | 'b', Box6<number>>()
	store.set('a', other).set('b', make6(other.value + 5))
	const mapped = make5(other.value)
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
