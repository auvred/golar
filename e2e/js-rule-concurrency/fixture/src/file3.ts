import { make4, type Box4 } from './file4.ts'

export interface Box3<T> {
	value: T
	map<U>(fn: (value: T) => U): Box3<U>
}

export function make3<T>(value: T): Box3<T> {
	return { value, map: (fn) => make3(fn(value)) }
}

export class Store3<K extends string, V> {
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

export function use3(other: Box4<number>) {
	const store = new Store3<'a' | 'b', Box4<number>>()
	store.set('a', other).set('b', make4(other.value + 3))
	const mapped = make3(other.value)
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
