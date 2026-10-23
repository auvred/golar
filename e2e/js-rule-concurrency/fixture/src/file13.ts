import { make14, type Box14 } from './file14.ts'

export interface Box13<T> {
	value: T
	map<U>(fn: (value: T) => U): Box13<U>
}

export function make13<T>(value: T): Box13<T> {
	return { value, map: (fn) => make13(fn(value)) }
}

export class Store13<K extends string, V> {
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

export function use13(other: Box14<number>) {
	const store = new Store13<'a' | 'b', Box14<number>>()
	store.set('a', other).set('b', make14(other.value + 13))
	const mapped = make13(other.value)
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
