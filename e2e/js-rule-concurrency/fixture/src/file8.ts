import { make9, type Box9 } from './file9.ts'

export interface Box8<T> {
	value: T
	map<U>(fn: (value: T) => U): Box8<U>
}

export function make8<T>(value: T): Box8<T> {
	return { value, map: (fn) => make8(fn(value)) }
}

export class Store8<K extends string, V> {
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

export function use8(other: Box9<number>) {
	const store = new Store8<'a' | 'b', Box9<number>>()
	store.set('a', other).set('b', make9(other.value + 8))
	const mapped = make8(other.value)
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
