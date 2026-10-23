import { make11, type Box11 } from './file11.ts'

export interface Box10<T> {
	value: T
	map<U>(fn: (value: T) => U): Box10<U>
}

export function make10<T>(value: T): Box10<T> {
	return { value, map: (fn) => make10(fn(value)) }
}

export class Store10<K extends string, V> {
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

export function use10(other: Box11<number>) {
	const store = new Store10<'a' | 'b', Box11<number>>()
	store.set('a', other).set('b', make11(other.value + 10))
	const mapped = make10(other.value)
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
