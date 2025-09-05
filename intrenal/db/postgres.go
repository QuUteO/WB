package db

import (
	"WB_Service/intrenal/lib/sl"
	model "WB_Service/intrenal/models"
	"context"
	"errors"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"log/slog"
)

type Postgres struct {
	pool *pgxpool.Pool
	log  *slog.Logger
}

type PostgresConfig struct {
	Host     string `yaml:"postgres_host" env-default:"postgres"`
	Username string `yaml:"postgres_user" env-default:"user"`
	Password string `yaml:"postgres_password" env-default:"1234" env-required:"true"`
	Database string `yaml:"postgres_db" env-default:"postgres"`
	Port     int    `yaml:"postgres_port" env-default:"5432"`

	MaxCon int `yaml:"postgres_max_conn" env-default:"10"`
	MinCon int `yaml:"postgres_min_conn" env-default:"5"`
}

func NewPostgres(ctx context.Context, config PostgresConfig, log *slog.Logger) (*Postgres, error) {
	conString := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?pool_max_conns=%d&pool_min_conns=%d&sslmode=disable",
		config.Username,
		config.Password,
		config.Host,
		config.Port,
		config.Database,
		config.MaxCon,
		config.MinCon,
	)

	conn, err := pgxpool.New(ctx, conString)
	if err != nil {
		slog.Error("Failed to connect to postgres", sl.Err(err))
		return nil, err
	}

	// Применям миграции
	m, err := migrate.New(
		"file://./db/migrations",
		fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
			config.Username,
			config.Password,
			config.Host,
			config.Port,
			config.Database,
		),
	)
	if err != nil {
		log.Error("Postgres migration error", sl.Err(err))
		return nil, err
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return nil, fmt.Errorf("failed to migrate to database: %w", err)
	}

	return &Postgres{
		pool: conn,
		log:  log,
	}, nil
}

func (p *Postgres) SaveUserData(ctx context.Context, order *model.Order) error {
	if p.pool == nil {
		return fmt.Errorf("pool is nil")
	}

	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func(tx pgx.Tx, ctx context.Context) {
		err := tx.Rollback(ctx)
		if err != nil {
			p.log.Error("Rollback failed", sl.Err(err))
		}
	}(tx, ctx)

	// сохраняем orders
	_, err = tx.Exec(
		ctx,
		`INSERT INTO orders (order_uid, 
                    track_number, 
                    entry, 
                    locale, 
                    internal_signature, 
                    customer_id, 
                    delivery_service, 
                    shardkey, 
                    sm_id, 
                    date_created, 
                    oof_shard) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
                    ON CONFLICT (order_uid) DO UPDATE SET track_number=EXCLUDED.track_number,
                                                          entry=EXCLUDED.entry,
                                                          locale=EXCLUDED.locale,
                                                          internal_signature=EXCLUDED.internal_signature,
                                                          customer_id=EXCLUDED.customer_id,
                                                          delivery_service=EXCLUDED.delivery_service,
                                                          shardkey=EXCLUDED.shardkey,
                                                          sm_id=EXCLUDED.sm_id,
                                                          date_created=EXCLUDED.date_created,
                                                          oof_shard=EXCLUDED.oof_shard`,
		order.OrderUUID, order.TrackNumber, order.Entry, order.Locale, order.InternalSignature,
		order.CustomerID, order.DeliveryService, order.Shardkey, order.SmID, order.DateCreated, order.OOFShard,
	)
	if err != nil {
		return fmt.Errorf("failed to save order: %w", err)
	}

	// сохраняем delivery
	_, err = tx.Exec(ctx, `INSERT INTO delivery (order_uid, name, phone, zip, city, address, region, email)  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (order_uid) DO UPDATE SET name=EXCLUDED.name,
                                      phone=EXCLUDED.phone,
                                      zip=EXCLUDED.zip,
                                      city=EXCLUDED.city,
                                      address=EXCLUDED.address,
                                      region=EXCLUDED.region,
                                      email=EXCLUDED.email`,
		order.OrderUUID, order.Delivery.Name, order.Delivery.Phone, order.Delivery.Zip, order.Delivery.City,
		order.Delivery.Address, order.Delivery.Region, order.Delivery.Email,
	)
	if err != nil {
		return fmt.Errorf("failed to save delivery: %w", err)
	}

	// сохраняем payment
	_, err = tx.Exec(ctx,
		`INSERT INTO payment (order_uid, transaction, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
        ON CONFLICT (order_uid) DO UPDATE SET
                                              transaction=EXCLUDED.transaction,
                                              request_id=EXCLUDED.request_id,
                                              currency=EXCLUDED.currency,
                                              provider=EXCLUDED.provider,
                                              amount=EXCLUDED.amount,
                                              payment_dt=EXCLUDED.payment_dt,
                                              bank=EXCLUDED.bank,
                                              delivery_cost=EXCLUDED.delivery_cost,
                                              goods_total=EXCLUDED.goods_total,
                                              custom_fee=EXCLUDED.custom_fee
                                              `,
		order.OrderUUID, order.Payment.Transaction, order.Payment.RequestId, order.Payment.Currency, order.Payment.Provider,
		order.Payment.Amount, order.Payment.Payment, order.Payment.Bank, order.Payment.DeliveryCost, order.Payment.GoodsTotal,
		order.Payment.CustomFee)
	if err != nil {
		return fmt.Errorf("failed to save payment: %w", err)
	}

	// сохраняем items
	for _, item := range order.Items {
		_, err = tx.Exec(ctx,
			`INSERT INTO items (order_uid, chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
			order.OrderUUID, item.ChrtID, item.TrackNumber, item.Price, item.Rid, item.Name, item.Sale, item.Size, item.TotalPrice, item.NmID, item.Brand, item.Status,
		)
	}

	return tx.Commit(ctx)
}

func (p *Postgres) GetOrder(ctx context.Context, orderUID string) (*model.Order, error) {
	if p.pool == nil {
		return nil, fmt.Errorf("pool is nil")
	}

	var order model.Order

	// получаем Orders
	err := p.pool.QueryRow(ctx, `SELECT order_uid, track_number, entry, locale, internal_signature, customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard FROM orders WHERE order_uid = $1`,
		orderUID).Scan(&order.OrderUUID,
		&order.TrackNumber,
		&order.Entry,
		&order.Locale,
		&order.InternalSignature,
		&order.CustomerID,
		&order.DeliveryService,
		&order.Shardkey,
		&order.SmID,
		&order.DateCreated,
		&order.OOFShard,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	// получаем Delivery
	err = p.pool.QueryRow(ctx, `SELECT name, phone, zip, city, address, region, email FROM delivery WHERE order_uid = $1`,
		orderUID).Scan(&order.Delivery.Name,
		&order.Delivery.Phone,
		&order.Delivery.Zip,
		&order.Delivery.City,
		&order.Delivery.Address,
		&order.Delivery.Region,
		&order.Delivery.Email,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	// получаем Payment
	err = p.pool.QueryRow(ctx,
		`SELECT transaction, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee FROM payment WHERE order_uid = $1`,
		orderUID).Scan(&order.Payment.Transaction,
		&order.Payment.RequestId,
		&order.Payment.Currency,
		&order.Payment.Provider,
		&order.Payment.Amount,
		&order.Payment.Payment,
		&order.Payment.Bank,
		&order.Payment.DeliveryCost,
		&order.Payment.GoodsTotal,
		&order.Payment.CustomFee,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	// получаем Items
	rows, err := p.pool.Query(ctx,
		`SELECT chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status FROM items WHERE order_uid = $1`,
		orderUID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item model.Item
		err = rows.Scan(&item.ChrtID,
			&item.TrackNumber,
			&item.Price,
			&item.Rid,
			&item.Name,
			&item.Sale,
			&item.Size,
			&item.TotalPrice,
			&item.NmID,
			&item.Brand,
			&item.Status,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to get order: %w", err)
		}
		order.Items = append(order.Items, item)
	}

	return &order, nil
}

func (p *Postgres) GetOrders(ctx context.Context) (map[string]*model.Order, error) {
	if p.pool == nil {
		return nil, fmt.Errorf("pool is nil")
	}

	// Берём только order_uid, чтобы потом подтянуть весь заказ через GetOrder
	rows, err := p.pool.Query(ctx, `SELECT order_uid FROM orders ORDER BY order_uid`)
	if err != nil {
		return nil, fmt.Errorf("failed to get all order_uids: %w", err)
	}
	defer rows.Close()

	orders := make(map[string]*model.Order)

	for rows.Next() {
		var orderUID string

		if err := rows.Scan(&orderUID); err != nil {
			return nil, fmt.Errorf("failed to scan order_uid: %w", err)
		}

		// Загружаем полный заказ по UID
		order, err := p.GetOrder(ctx, orderUID)
		if err != nil {
			return nil, fmt.Errorf("failed to get order %s: %w", orderUID, err)
		}

		orders[orderUID] = order
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return orders, nil
}
