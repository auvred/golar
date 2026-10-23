import { make8, type Box8 } from './file8.ts'

export interface Box7<T> {
	value: T
	map<U>(fn: (value: T) => U): Box7<U>
}

export function make7<T>(value: T): Box7<T> {
	return { value, map: (fn) => make7(fn(value)) }
}

export class Store7<K extends string, V> {
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

export function use7(other: Box8<number>) {
	const store = new Store7<'a' | 'b', Box8<number>>()
	store.set('a', other).set('b', make8(other.value + 7))
	const mapped = make7(other.value)
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
