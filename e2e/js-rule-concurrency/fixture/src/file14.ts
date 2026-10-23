import { make15, type Box15 } from './file15.ts'

export interface Box14<T> {
	value: T
	map<U>(fn: (value: T) => U): Box14<U>
}

export function make14<T>(value: T): Box14<T> {
	return { value, map: (fn) => make14(fn(value)) }
}

export class Store14<K extends string, V> {
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

export function use14(other: Box15<number>) {
	const store = new Store14<'a' | 'b', Box15<number>>()
	store.set('a', other).set('b', make15(other.value + 14))
	const mapped = make14(other.value)
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
