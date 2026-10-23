import { make7, type Box7 } from './file7.ts'

export interface Box6<T> {
	value: T
	map<U>(fn: (value: T) => U): Box6<U>
}

export function make6<T>(value: T): Box6<T> {
	return { value, map: (fn) => make6(fn(value)) }
}

export class Store6<K extends string, V> {
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

export function use6(other: Box7<number>) {
	const store = new Store6<'a' | 'b', Box7<number>>()
	store.set('a', other).set('b', make7(other.value + 6))
	const mapped = make6(other.value)
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
