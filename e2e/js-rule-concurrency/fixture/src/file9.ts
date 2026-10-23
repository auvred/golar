import { make10, type Box10 } from './file10.ts'

export interface Box9<T> {
	value: T
	map<U>(fn: (value: T) => U): Box9<U>
}

export function make9<T>(value: T): Box9<T> {
	return { value, map: (fn) => make9(fn(value)) }
}

export class Store9<K extends string, V> {
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

export function use9(other: Box10<number>) {
	const store = new Store9<'a' | 'b', Box10<number>>()
	store.set('a', other).set('b', make10(other.value + 9))
	const mapped = make9(other.value)
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
