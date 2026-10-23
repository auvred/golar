import { make3, type Box3 } from './file3.ts'

export interface Box2<T> {
	value: T
	map<U>(fn: (value: T) => U): Box2<U>
}

export function make2<T>(value: T): Box2<T> {
	return { value, map: (fn) => make2(fn(value)) }
}

export class Store2<K extends string, V> {
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

export function use2(other: Box3<number>) {
	const store = new Store2<'a' | 'b', Box3<number>>()
	store.set('a', other).set('b', make3(other.value + 2))
	const mapped = make2(other.value)
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
